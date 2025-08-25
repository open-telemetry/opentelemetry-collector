// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pref // import "go.opentelemetry.io/collector/pdata/xpdata/pref"

import (
	"reflect"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MarkPipelineOwnedMetrics marks the pmetric.Metrics data as owned by the pipeline, returns true if the data were
// previously not owned by the pipeline, otherwise false.
func MarkPipelineOwnedMetrics(md pmetric.Metrics) bool {
	return internal.GetMetricsState(internal.Metrics(md)).MarkPipelineOwned()
}

func RefMetrics(md pmetric.Metrics) {
	if EnableRefCounting.IsEnabled() {
		internal.GetMetricsState(internal.Metrics(md)).Ref()
	}
}

func UnrefMetrics(md pmetric.Metrics) {
	if EnableRefCounting.IsEnabled() {
		if !internal.GetMetricsState(internal.Metrics(md)).Unref() {
			return
		}
		// Don't call DeleteOrigExportLogsServiceRequest without the gate because we reset the data and that may still cause issues.
		if internal.UseProtoPooling.IsEnabled() {
			internal.DeleteOrigExportMetricsServiceRequest(internal.GetOrigMetrics(internal.Metrics(md)), true)
		}
	}
}

// TODO: Generate this in pdata.

func EqualMetrics(md1, md2 pmetric.Metrics) bool {
	return reflect.DeepEqual(internal.GetOrigMetrics(internal.Metrics(md1)), internal.GetOrigMetrics(internal.Metrics(md2)))
}
