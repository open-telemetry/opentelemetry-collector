// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package exporterhelper defines internal capabilities of exporters built with exporterhelper.
package exporterhelper // import "go.opentelemetry.io/collector/internal/exporterhelper"

// ExporterHelper exposes internal properties of an exporter built with exporterhelper.
type ExporterHelper interface {
	BatchingEnabled() bool
	private()
}

// NewExporterHelper returns an ExporterHelper with the given batching state.
func NewExporterHelper(batchingEnabled bool) ExporterHelper {
	return exporter{batchingEnabled: batchingEnabled}
}

type exporter struct {
	batchingEnabled bool
}

func (e exporter) BatchingEnabled() bool {
	return e.batchingEnabled
}

func (exporter) private() {}
