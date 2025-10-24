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
	return internal.GetMetricsState(internal.MetricsWrapper(md)).MarkPipelineOwned()
}

func RefMetrics(md pmetric.Metrics) {
	if EnableRefCounting.IsEnabled() {
		internal.GetMetricsState(internal.MetricsWrapper(md)).Ref()
	}
}

func UnrefMetrics(md pmetric.Metrics) {
	if EnableRefCounting.IsEnabled() {
		if !internal.GetMetricsState(internal.MetricsWrapper(md)).Unref() {
			return
		}
		// Don't call DeleteExportLogsServiceRequest without the gate because we reset the data and that may still cause issues.
		if internal.UseProtoPooling.IsEnabled() {
			internal.DeleteExportMetricsServiceRequest(internal.GetMetricsOrig(internal.MetricsWrapper(md)), true)
		}
	}
}

// TODO: Generate this in pdata.

func EqualMetrics(md1, md2 pmetric.Metrics) bool {
	return reflect.DeepEqual(internal.GetMetricsOrig(internal.MetricsWrapper(md1)), internal.GetMetricsOrig(internal.MetricsWrapper(md2)))
}
