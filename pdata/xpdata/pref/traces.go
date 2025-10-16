// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pref // import "go.opentelemetry.io/collector/pdata/xpdata/pref"

import (
	"reflect"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// MarkPipelineOwnedTraces marks the ptrace.Traces data as owned by the pipeline, returns true if the data were
// previously not owned by the pipeline, otherwise false.
func MarkPipelineOwnedTraces(td ptrace.Traces) bool {
	return internal.GetTracesState(internal.Traces(td)).MarkPipelineOwned()
}

func RefTraces(td ptrace.Traces) {
	internal.GetTracesState(internal.Traces(td)).Ref()
}

func UnrefTraces(td ptrace.Traces) {
	if EnableRefCounting.IsEnabled() {
		if !internal.GetTracesState(internal.Traces(td)).Unref() {
			return
		}
		// Don't call DeleteOrigExportLogsServiceRequest without the gate because we reset the data and that may still cause issues.
		if internal.UseProtoPooling.IsEnabled() {
			internal.DeleteOrigExportTraceServiceRequest(internal.GetOrigTraces(internal.Traces(td)), true)
		}
	}
}

// TODO: Generate this in pdata.

func EqualTraces(td1, td2 ptrace.Traces) bool {
	return reflect.DeepEqual(internal.GetOrigTraces(internal.Traces(td1)), internal.GetOrigTraces(internal.Traces(td2)))
}
