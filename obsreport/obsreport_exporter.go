// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import "go.opentelemetry.io/collector/exporter/exporterhelper"

// Exporter is a helper to add observability to an exporter.
//
// Deprecated: [0.85.0] Use exporterhelper.Exporter instead.
type Exporter = exporterhelper.Exporter

// ExporterSettings are settings for creating an Exporter.
//
// Deprecated: [0.85.0] Use exporterhelper.Settings instead.
type ExporterSettings = exporterhelper.Settings

// NewExporter creates a new Exporter.
//
// Deprecated: [0.85.0] Use exporterhelper.New instead.
func NewExporter(cfg ExporterSettings) (*exporterhelper.Exporter, error) {
	return exporterhelper.New(exporterhelper.Settings{
		ExporterID:             cfg.ExporterID,
		ExporterCreateSettings: cfg.ExporterCreateSettings,
	})
}
