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

type diskIoScraper struct{}

func newDiskIoScraper() *diskIoScraper {
	return &diskIoScraper{}
}

func (s *diskIoScraper) Scrape(logger *zap.Logger, meta *Metadata) ([]pmetric.Metric, error) {
	// 1. First Sample
	s1, err := s.readDiskStats()
	if err != nil {
		return nil, fmt.Errorf("diskio sample 1 failed: %w", err)
	}

	// 2. Wait 1s
	time.Sleep(1 * time.Second)

	// 3. Second Sample
	s2, err := s.readDiskStats()
	if err != nil {
		return nil, fmt.Errorf("diskio sample 2 failed: %w", err)
	}

	metrics := []pmetric.Metric{}

	now := pcommon.NewTimestampFromTime(time.Now())

	// Iterate over devices present in both samples
	for dev, st2 := range s2 {
		st1, ok := s1[dev]
		if !ok {
			continue
		}

		// Calculations
		// 1. IO Read (KB) = (SectorsDiff * 512) / 1024 = SectorsDiff / 2
		diffReadKeep := float64(st2.sectorsRead-st1.sectorsRead) / 2.0

		// 2. IO Write (KB)
		diffWriteKeep := float64(st2.sectorsWritten-st1.sectorsWritten) / 2.0

		// 3. Rate (Util %) = (IoTimeDiffMs) / 1000
		// User formula: (stat2.IoTime-stat.IoTime)/1000
		// If IoTime is ms. Diff is ms. /1000 = Seconds.
		// If utilization, usually * 100.
		// But user asked for Rate.

		diffTime := float64(st2.ioTime - st1.ioTime)
		rate := diffTime / 1000.0

		// Generate 3 Metrics: Rate, RDKB, WTKB

		// Metric 1: RATE
		mRate := pmetric.NewMetric()
		mRate.SetName("sys_disk_io_rate")
		mRate.SetEmptyGauge()
		dp := mRate.Gauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(rate)
		setCommonAttributes(dp, now, meta, dev)
		metrics = append(metrics, mRate)

		// Metric 2: RDKB -> read_kb
		mRead := pmetric.NewMetric()
		mRead.SetName("sys_disk_io_read_kb")
		mRead.SetEmptyGauge()
		dpRead := mRead.Gauge().DataPoints().AppendEmpty()
		dpRead.SetDoubleValue(diffReadKeep)
		setCommonAttributes(dpRead, now, meta, dev)
		metrics = append(metrics, mRead)

		// Metric 3: WTKB -> write_kb
		mWrite := pmetric.NewMetric()
		mWrite.SetName("sys_disk_io_write_kb")
		mWrite.SetEmptyGauge()
		dpWrite := mWrite.Gauge().DataPoints().AppendEmpty()
		dpWrite.SetDoubleValue(diffWriteKeep)
		setCommonAttributes(dpWrite, now, meta, dev)
		metrics = append(metrics, mWrite)
	}

	return metrics, nil
}

func setCommonAttributes(dp pmetric.NumberDataPoint, now pcommon.Timestamp, meta *Metadata, device string) {
	dp.SetTimestamp(now)
	dp.Attributes().PutStr(meta.NodeColumnName, meta.NodeValue)
	dp.Attributes().PutStr("HOST_IP", meta.HostIP)
	dp.Attributes().PutStr("DEVICE", device)
}

type startInfo struct {
	sectorsRead    uint64
	sectorsWritten uint64
	ioTime         uint64
}

func (s *diskIoScraper) readDiskStats() (map[string]startInfo, error) {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return nil, err
	}

	stats := make(map[string]startInfo)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		devName := fields[2]
		// Filter: sd*, vd*, nvme*
		if !strings.HasPrefix(devName, "sd") && !strings.HasPrefix(devName, "vd") && !strings.HasPrefix(devName, "nvme") {
			continue
		}

		// Field 6 (index 5): Sectors Read
		secRead, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			continue
		}
		// Field 10 (index 9): Sectors Written
		secWrite, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}
		// Field 13 (index 12): Time spend doing IO (ms)
		ioTime, err := strconv.ParseUint(fields[12], 10, 64)
		if err != nil {
			continue
		}

		stats[devName] = startInfo{
			sectorsRead:    secRead,
			sectorsWritten: secWrite,
			ioTime:         ioTime,
		}
	}
	return stats, nil
}
