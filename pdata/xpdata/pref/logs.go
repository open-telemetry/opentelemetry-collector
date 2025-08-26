// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pref // import "go.opentelemetry.io/collector/pdata/xpdata/pref"

import (
	"reflect"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/plog"
)

// MarkPipelineOwnedLogs marks the plog.Logs data as owned by the pipeline, returns true if the data were
// previously not owned by the pipeline, otherwise false.
func MarkPipelineOwnedLogs(ld plog.Logs) bool {
	return internal.GetLogsState(internal.Logs(ld)).MarkPipelineOwned()
}

func RefLogs(ld plog.Logs) {
	if EnableRefCounting.IsEnabled() {
		internal.GetLogsState(internal.Logs(ld)).Ref()
	}
}

func UnrefLogs(ld plog.Logs) {
	if EnableRefCounting.IsEnabled() {
		if !internal.GetLogsState(internal.Logs(ld)).Unref() {
			return
		}
		// Don't call DeleteOrigExportLogsServiceRequest without the gate because we reset the data and that may still cause issues.
		if internal.UseProtoPooling.IsEnabled() {
			internal.DeleteOrigExportLogsServiceRequest(internal.GetOrigLogs(internal.Logs(ld)), true)
		}
	}
}

// TODO: Generate this in pdata.

func EqualLogs(ld1, ld2 plog.Logs) bool {
	return reflect.DeepEqual(internal.GetOrigLogs(internal.Logs(ld1)), internal.GetOrigLogs(internal.Logs(ld2)))
}
