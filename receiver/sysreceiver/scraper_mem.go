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

type memScraper struct{}

func newMemScraper() *memScraper {
	return &memScraper{}
}

func (s *memScraper) Scrape(logger *zap.Logger, meta *Metadata) ([]pmetric.Metric, error) {
	// Read /proc/meminfo
	memStats, err := s.readMemStats()
	if err != nil {
		return nil, fmt.Errorf("failed to read mem info: %w", err)
	}

	// Formula:
	// Used = Total - Free - Buffers - Cached
	// Rate = Used / Total * 100

	total := memStats["MemTotal"]
	free := memStats["MemFree"]
	buffers := memStats["Buffers"]
	cached := memStats["Cached"]
	// SReclaimable is often counted as cached, but user formula didn't explicitly ask.
	// We strictly follow: Total - Free - Buffers - Cached.

	used := total - free - buffers - cached
	rate := 0.0
	if total > 0 {
		rate = (float64(used) / float64(total)) * 100
	}

	// Table Name: sys_memory_usage (Static)
	tableName := "sys_memory_usage"

	m := pmetric.NewMetric()
	m.SetName(tableName)
	m.SetDescription("Memory Usage Rate")
	m.SetEmptyGauge()

	dp := m.Gauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(rate)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	dp.Attributes().PutStr(meta.NodeColumnName, meta.NodeValue)
	dp.Attributes().PutStr("HOST_IP", meta.HostIP)

	return []pmetric.Metric{m}, nil
}

func (s *memScraper) readMemStats() (map[string]uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	stats := make(map[string]uint64)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}
		// Unit is usually kB, but since we calculate ratio, units cancel out.
		stats[key] = val
	}
	return stats, nil
}
