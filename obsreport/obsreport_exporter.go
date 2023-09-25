// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import "go.opentelemetry.io/collector/exporter/exporterhelper"

// Exporter is a helper to add observability to an exporter.
//
// Deprecated: [0.86.0] Use exporterhelper.ObsReport instead.
type Exporter = exporterhelper.ObsReport

// ExporterSettings are settings for creating an Exporter.
//
// Deprecated: [0.86.0] Use exporterhelper.ObsReportSettings instead.
type ExporterSettings = exporterhelper.ObsReportSettings

// NewExporter creates a new Exporter.
//
// Deprecated: [0.86.0] Use exporterhelper.New instead.
func NewExporter(cfg ExporterSettings) (*exporterhelper.ObsReport, error) {
	return exporterhelper.NewObsReport(cfg)
}
