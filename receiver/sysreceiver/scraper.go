package sysreceiver

import (
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// Scraper defines the interface for system metric scrapers.
type Scraper interface {
	// Scrape collects metrics and appends them to the provided slice.
	// It returns the number of metrics added and any error encountered.
	Scrape(logger *zap.Logger, meta *Metadata) ([]pmetric.Metric, error)
}

// Metadata holds the common identity information for all metrics.
type Metadata struct {
	NodeValue          string
	NodeColumnName     string
	HostIP             string
	DeploymentInstance string
}
