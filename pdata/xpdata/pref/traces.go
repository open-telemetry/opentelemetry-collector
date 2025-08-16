// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pref

import (
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
	if !internal.GetTracesState(internal.Traces(td)).Unref() {
		return
	}
	internal.ReleaseOrigExportTraceServiceRequest(internal.GetOrigTraces(internal.Traces(td)))
}
