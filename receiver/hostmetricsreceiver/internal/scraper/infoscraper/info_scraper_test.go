package infoscraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

func TestScrape(t *testing.T) {
	scraper, err := newInfoScraper(context.Background())
	scraper.org = func() string {
		return "8888"
	}
	scraper.cpuNum = func() int {
		return 8
	}
	scraper.hostname = func() (name string, err error) {
		return "aaronhedeMacBook-Pro.local", nil
	}
	assert.Nil(t, err)
	metrics, err := scraper.Scrape(context.Background())
	assert.Equal(t, metrics.Len(), 1)
	metric := metrics.At(0)
	internal.AssertIntSumMetricLabelHasValue(t, metric, 0, "hostname",  "aaronhedeMacBook-Pro.local")
	internal.AssertIntSumMetricLabelHasValue(t, metric, 0, "cpuNum", "8")
	internal.AssertIntSumMetricLabelHasValue(t, metric, 0, "org", "8888")
}
