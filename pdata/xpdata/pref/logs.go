// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pref // import "go.opentelemetry.io/collector/pdata/xpdata/pref"

import (
	"reflect"

	"go.opentelemetry.io/collector/pdata/internal"
	pmetadata "go.opentelemetry.io/collector/pdata/internal/metadata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/xpdata/internal/metadata"
)

// MarkPipelineOwnedLogs marks the plog.Logs data as owned by the pipeline, returns true if the data were
// previously not owned by the pipeline, otherwise false.
func MarkPipelineOwnedLogs(ld plog.Logs) bool {
	return internal.GetLogsState(internal.LogsWrapper(ld)).MarkPipelineOwned()
}

func RefLogs(ld plog.Logs) {
	if metadata.PdataEnableRefCountingFeatureGate.IsEnabled() {
		internal.GetLogsState(internal.LogsWrapper(ld)).Ref()
	}
}

func UnrefLogs(ld plog.Logs) {
	if metadata.PdataEnableRefCountingFeatureGate.IsEnabled() {
		if !internal.GetLogsState(internal.LogsWrapper(ld)).Unref() {
			return
		}
		// Don't call DeleteExportLogsServiceRequest without the gate because we reset the data and that may still cause issues.
		if pmetadata.PdataUseProtoPoolingFeatureGate.IsEnabled() {
			internal.DeleteExportLogsServiceRequest(internal.GetLogsOrig(internal.LogsWrapper(ld)), true)
		}
	}
}

// TODO: Generate this in pdata.

func EqualLogs(ld1, ld2 plog.Logs) bool {
	return reflect.DeepEqual(internal.GetLogsOrig(internal.LogsWrapper(ld1)), internal.GetLogsOrig(internal.LogsWrapper(ld2)))
}
