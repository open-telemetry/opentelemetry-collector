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

type cpuScraper struct{}

func newCpuScraper() *cpuScraper {
	return &cpuScraper{}
}

func (s *cpuScraper) Scrape(logger *zap.Logger, meta *Metadata) ([]pmetric.Metric, error) {
	// 1. First Sample
	t1, b1, err := s.readCPUStats()
	if err != nil {
		return nil, fmt.Errorf("failed to read cpu stats (1st sample): %w", err)
	}

	// 2. Wait 1 second
	time.Sleep(1 * time.Second)

	// 3. Second Sample
	t2, b2, err := s.readCPUStats()
	if err != nil {
		return nil, fmt.Errorf("failed to read cpu stats (2nd sample): %w", err)
	}

	// 4. Calculate Rate
	diffTotal := t2 - t1
	diffBusy := b2 - b1

	usageRate := 0.0
	if diffTotal > 0 {
		usageRate = (float64(diffBusy) / float64(diffTotal)) * 100
	}

	// 5. Construct Metric
	// Table Name: sys_cpu_usage (Static)
	tableName := "sys_cpu_usage"

	m := pmetric.NewMetric()
	m.SetName(tableName)
	m.SetDescription("CPU Usage Rate")
	m.SetEmptyGauge()

	dp := m.Gauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(usageRate)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Attributes
	dp.Attributes().PutStr(meta.NodeColumnName, meta.NodeValue)
	dp.Attributes().PutStr("HOST_IP", meta.HostIP)

	return []pmetric.Metric{m}, nil
}

func (s *cpuScraper) readCPUStats() (uint64, uint64, error) {
	const procStatFile = "/proc/stat"

	// Read file
	data, err := os.ReadFile(procStatFile)
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Look for "cpu" line (aggregated)
		if fields[0] == "cpu" {
			// Fields:
			// 0: cpu
			// 1: user, 2: nice, 3: system, 4: idle, 5: iowait, 6: irq, 7: softirq, 8: steal, 9: guest, 10: guest_nice
			// Parsed Numbers start at index 1

			var values []uint64
			for i := 1; i < len(fields); i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					continue
				}
				values = append(values, val)
			}

			if len(values) < 8 {
				return 0, 0, fmt.Errorf("not enough fields in cpu line")
			}

			// User formula:
			// total = User + System + Idle + Nice + Iowait + Irq + Softirq + Steal
			// Indices in 'values':
			// user=0, nice=1, system=2, idle=3, iowait=4, irq=5, softirq=6, steal=7

			user := values[0]
			nice := values[1]
			system := values[2]
			idle := values[3]
			iowait := values[4]
			irq := values[5]
			softirq := values[6]
			steal := values[7]

			total := user + nice + system + idle + iowait + irq + softirq + steal
			busy := total - idle - iowait

			return total, busy, nil
		}
	}

	return 0, 0, fmt.Errorf("cpu line not found in %s", procStatFile)
}
