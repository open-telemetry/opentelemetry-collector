// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"

	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// TODO: Incorporate this functionality along with tests from obsreport_test.go
//       into existing `obsreport` package once its functionally is not exposed
//       as public API. For now this part is kept private.

var (
	globalInstruments = newInstruments(metric.NewRegistry())
)

func init() {
	metricproducer.GlobalManager().AddProducer(globalInstruments.registry)
}

type instruments struct {
	registry      *metric.Registry
	queueSize     *metric.Int64DerivedGauge
	queueCapacity *metric.Int64DerivedGauge
}

func newInstruments(registry *metric.Registry) *instruments {
	insts := &instruments{
		registry: registry,
	}
	insts.queueSize, _ = registry.AddInt64DerivedGauge(
		obsmetrics.ExporterKey+"/queue_size",
		metric.WithDescription("Current size of the retry queue (in batches)"),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	insts.queueCapacity, _ = registry.AddInt64DerivedGauge(
		obsmetrics.ExporterKey+"/queue_capacity",
		metric.WithDescription("Fixed capacity of the retry queue (in batches)"),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))
	return insts
}
