// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	metricPrefix = "otelcol_http_server"

	defaultMaxHttpConnectionsMetricName = metricPrefix + "http_max_connections"
	defaultHttpConnectionsMetricName    = metricPrefix + "http_connections"
)

// HttpServerTelemetryConfig defines the telemetry configuration for HTTPServerSettings
type HttpServerTelemetryConfig struct {
	// Enabled is set to enable opencensus metrics
	Enabled bool `mapstructure:"enabled"`
	// ExtraAttributes is the extra attributes for metrics
	ExtraAttributes map[string]string `mapstructure:"extra_attributes"`
}

// httpServerTelemetryInstruments is a helper to add observability to http server.
type httpServerTelemetryInstruments struct {
	// conn is a gauge measuring connections to the http server
	conn *stats.Int64Measure
	// maxConn is a static value indicating max connections for the http server
	maxConn *stats.Int64Measure
	// tags is dimensions for the associated metrics
	tags []tag.Mutator
}

// newHttpServerTelemetryInstruments creates observability for HTTPServerSettings.
// Currently, only supports opencensus metrics, but can extend to otel metrics
func newHttpServerTelemetryInstruments(c HttpServerTelemetryConfig) *httpServerTelemetryInstruments {
	var tags []tag.Key
	var tagMutators []tag.Mutator
	var maxConnMeasure, connMeasure *stats.Int64Measure
	for key, val := range c.ExtraAttributes {
		newTag, _ := tag.NewKey(key)
		tags = append(tags, newTag)
		tagMutators = append(tagMutators, tag.Insert(newTag, val))
	}

	connMeasure = stats.Int64(
		defaultHttpConnectionsMetricName,
		"number of connections being handled",
		stats.UnitDimensionless)
	connView := &view.View{
		Name:        connMeasure.Name(),
		Description: connMeasure.Description(),
		TagKeys:     tags,
		Measure:     connMeasure,
		Aggregation: view.LastValue(),
	}

	maxConnMeasure = stats.Int64(
		defaultMaxHttpConnectionsMetricName,
		"maximum numbers of connections",
		stats.UnitDimensionless)
	maxConnView := &view.View{
		Name:        maxConnMeasure.Name(),
		Description: maxConnMeasure.Description(),
		TagKeys:     tags,
		Measure:     maxConnMeasure,
		Aggregation: view.LastValue(),
	}

	if err := view.Register(connView, maxConnView); err != nil {
		return nil
	}

	return &httpServerTelemetryInstruments{
		conn:    connMeasure,
		maxConn: maxConnMeasure,
		tags:    tagMutators,
	}
}
