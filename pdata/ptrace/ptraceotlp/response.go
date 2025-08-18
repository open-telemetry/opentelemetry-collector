// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptraceotlp // import "go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

// MarshalProto marshals ExportResponse into proto bytes.
func (ms ExportResponse) MarshalProto() ([]byte, error) {
	if !internal.UseCustomProtoEncoding.IsEnabled() {
		return ms.orig.Marshal()
	}
	size := internal.SizeProtoOrigExportTraceServiceResponse(ms.orig)
	buf := make([]byte, size)
	_ = internal.MarshalProtoOrigExportTraceServiceResponse(ms.orig, buf)
	return buf, nil
}

// UnmarshalProto unmarshalls ExportResponse from proto bytes.
func (ms ExportResponse) UnmarshalProto(data []byte) error {
	if !internal.UseCustomProtoEncoding.IsEnabled() {
		return ms.orig.Unmarshal(data)
	}
	return internal.UnmarshalProtoOrigExportTraceServiceResponse(ms.orig, data)
}

// MarshalJSON marshals ExportResponse into JSON bytes.
func (ms ExportResponse) MarshalJSON() ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	internal.MarshalJSONOrigExportTraceServiceResponse(ms.orig, dest)
	return slices.Clone(dest.Buffer()), dest.Error()
}

// UnmarshalJSON unmarshalls ExportResponse from JSON bytes.
func (ms ExportResponse) UnmarshalJSON(data []byte) error {
	iter := json.BorrowIterator(data)
	defer json.ReturnIterator(iter)
	internal.UnmarshalJSONOrigExportTraceServiceResponse(ms.orig, iter)
	return iter.Error()
}
