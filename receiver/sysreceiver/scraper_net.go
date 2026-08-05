package sysreceiver

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type netScraper struct{}

func newNetScraper() *netScraper {
	return &netScraper{}
}

func (s *netScraper) Scrape(logger *zap.Logger, meta *Metadata) ([]pmetric.Metric, error) {
	// 1. First Sample
	rx1, tx1, err := s.readNetStats()
	if err != nil {
		return nil, fmt.Errorf("net sample 1 failed: %w", err)
	}

	// 2. Wait 1s
	time.Sleep(1 * time.Second)

	// 3. Second Sample
	rx2, tx2, err := s.readNetStats()
	if err != nil {
		return nil, fmt.Errorf("net sample 2 failed: %w", err)
	}

	// 4. Calculate Diff (KB)
	diffRxKB := float64(rx2-rx1) / 1024.0
	diffTxKB := float64(tx2-tx1) / 1024.0

	// 5. Construct Metrics
	now := pcommon.NewTimestampFromTime(time.Now())

	metrics := []pmetric.Metric{}

	// T_MON_NETIN -> sys_net_io_in
	mIn := pmetric.NewMetric()
	mIn.SetName("sys_net_io_in")
	mIn.SetEmptyGauge()
	dpIn := mIn.Gauge().DataPoints().AppendEmpty()
	dpIn.SetDoubleValue(diffRxKB)
	dpIn.SetTimestamp(now)
	dpIn.Attributes().PutStr(meta.NodeColumnName, meta.NodeValue)
	dpIn.Attributes().PutStr("HOST_IP", meta.HostIP)
	metrics = append(metrics, mIn)

	// T_MON_NETOUT -> sys_net_io_out
	mOut := pmetric.NewMetric()
	mOut.SetName("sys_net_io_out")
	mOut.SetEmptyGauge()
	dpOut := mOut.Gauge().DataPoints().AppendEmpty()
	dpOut.SetDoubleValue(diffTxKB)
	dpOut.SetTimestamp(now)
	dpOut.Attributes().PutStr(meta.NodeColumnName, meta.NodeValue)
	dpOut.Attributes().PutStr("HOST_IP", meta.HostIP)
	metrics = append(metrics, mOut)

	return metrics, nil
}

// readNetStats returns total RX bytes and total TX bytes (aggregated)
func (s *netScraper) readNetStats() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}

	var totalRx, totalTx uint64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		// skip headers
		if strings.Contains(line, "|") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		interfaceName := strings.TrimSuffix(parts[0], ":")
		if interfaceName == "lo" {
			continue
		}

		// Field indices depend on format. usually:
		// face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
		// 0    |1        2       3    4    5    6     7          8        |9        10      ...

		// parts[0] is name (sometimes "eth0:" sometimes "eth0" and parts[1] is bytes)
		// Standard parsing:
		var fields []string
		if strings.Contains(parts[0], ":") {
			// merged "eth0:1234"
			split := strings.SplitN(parts[0], ":", 2)
			if len(split[1]) > 0 {
				fields = append([]string{split[0]}, split[1])
				fields = append(fields, parts[1:]...)
			} else {
				fields = parts // just colon at end
			}
		} else {
			fields = parts
		}

		// After normalization:
		// 0: net
		// 1: rx_bytes
		// 9: tx_bytes

		if len(fields) < 10 {
			// Fallback logic if needed, but standard /proc/net/dev is strict
			continue
		}

		rx, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		tx, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}

		totalRx += rx
		totalTx += tx
	}
	return totalRx, totalTx, nil
}
