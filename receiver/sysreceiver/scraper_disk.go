//go:build linux

package sysreceiver

import (
	"fmt"
	"runtime"
	"syscall"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type diskScraper struct{}

func newDiskScraper() *diskScraper {
	return &diskScraper{}
}

func (s *diskScraper) Scrape(logger *zap.Logger, meta *Metadata) ([]pmetric.Metric, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("diskScraper is only supported on Linux")
	}

	path := "/"

	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return nil, fmt.Errorf("statfs failed for %s: %w", path, err)
	}

	total := uint64(fs.Blocks) * uint64(fs.Bsize)
	free := uint64(fs.Bfree) * uint64(fs.Bsize)
	used := total - free

	rate := 0.0
	if total > 0 {
		rate = (float64(used) / float64(total)) * 100
	}

	tableName := "sys_disk_usage"

	m := pmetric.NewMetric()
	m.SetName(tableName)
	m.SetDescription("Disk Usage Rate")
	m.SetEmptyGauge()

	dp := m.Gauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(rate)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	dp.Attributes().PutStr(meta.NodeColumnName, meta.NodeValue)
	dp.Attributes().PutStr("HOST_IP", meta.HostIP)

	return []pmetric.Metric{m}, nil
}
