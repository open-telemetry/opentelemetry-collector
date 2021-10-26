package infoscraper

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

const (
	standardMetricsLen = 1
	metricsLen         = standardMetricsLen
)

// scraper for Info Metrics
type scraper struct {
	hostname func() (name string, err error)
	now      func() time.Time
	cpuNum   func() int
	org      func() string
}

// newInfoScraper creates a info Scraper
func newInfoScraper(_ context.Context) (*scraper, error) {
	scraper := &scraper{
		hostname: os.Hostname,
		now:      time.Now,
		cpuNum:   runtime.NumCPU,
		org:      org,
	}
	return scraper, nil
}

// Scrape
func (s *scraper) Scrape(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	now := pdata.TimestampFromTime(time.Now())

	metrics.Resize(metricsLen)

	hostname, err := s.hostname()
	if err != nil {
		return metrics, err
	}
	metric := metrics.At(0)
	metadata.Metrics.InfoNow.Init(metric)

	idps := metric.IntSum().DataPoints()

	idps.Resize(1)

	initializeInfoDataPoint(idps.At(0), now, hostname, s.org(), s.cpuNum())

	return metrics, nil
}

func initializeInfoDataPoint(dataPoint pdata.IntDataPoint, now pdata.Timestamp, hostname string, org string, cpuNum int) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.InfoOrg, org)
	labelsMap.Insert(metadata.Labels.InfoHostname, hostname)
	labelsMap.Insert(metadata.Labels.InfoCPUNum, strconv.Itoa(cpuNum))
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(now.AsTime().Unix())
}

func org() string {
	return os.Getenv("EASY_ENV_COMMON_ORG")
}
