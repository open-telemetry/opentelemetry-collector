// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptraceotlp // import "go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

// ExportResponse represents the response for gRPC/HTTP client/server.
type ExportResponse struct {
	orig  *otlpcollectortrace.ExportTraceServiceResponse
	state *internal.State
}

// NewExportResponse returns an empty ExportResponse.
func NewExportResponse() ExportResponse {
	state := internal.StateMutable
	return ExportResponse{
		orig:  &otlpcollectortrace.ExportTraceServiceResponse{},
		state: &state,
	}
}

// MarshalProto marshals ExportResponse into proto bytes.
func (ms ExportResponse) MarshalProto() ([]byte, error) {
	return ms.orig.Marshal()
}

// UnmarshalProto unmarshalls ExportResponse from proto bytes.
func (ms ExportResponse) UnmarshalProto(data []byte) error {
	return ms.orig.Unmarshal(data)
}

// MarshalJSON marshals ExportResponse into JSON bytes.
func (ms ExportResponse) MarshalJSON() ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	dest.WriteObjectStart()
	dest.WriteObjectField("partialSuccess")
	internal.MarshalJSONOrigExportTracePartialSuccess(&ms.orig.PartialSuccess, dest)
	dest.WriteObjectEnd()
	return slices.Clone(dest.Buffer()), dest.Error()
}

// UnmarshalJSON unmarshalls ExportResponse from JSON bytes.
func (ms ExportResponse) UnmarshalJSON(data []byte) error {
	iter := json.BorrowIterator(data)
	defer json.ReturnIterator(iter)
	ms.unmarshalJSONIter(iter)
	return iter.Error()
}

func (ms ExportResponse) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "partial_success", "partialSuccess":
			internal.UnmarshalJSONOrigExportTracePartialSuccess(&ms.orig.PartialSuccess, iter)
		default:
			iter.Skip()
		}
		return true
	})
}

// PartialSuccess returns the ExportLogsPartialSuccess associated with this ExportResponse.
func (ms ExportResponse) PartialSuccess() ExportPartialSuccess {
	return newExportPartialSuccess(&ms.orig.PartialSuccess, ms.state)
}
