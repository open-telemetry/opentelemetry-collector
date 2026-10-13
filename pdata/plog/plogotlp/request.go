// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plogotlp // import "go.opentelemetry.io/collector/pdata/plog/plogotlp"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
	"go.opentelemetry.io/collector/pdata/plog"
)

// ExportRequest represents the request for gRPC/HTTP client/server.
// It's a wrapper for plog.Logs data.
type ExportRequest struct {
	orig  *internal.ExportLogsServiceRequest
	state *internal.State
}

// NewExportRequest returns an empty ExportRequest.
func NewExportRequest() ExportRequest {
	st := internal.NewState()
	return ExportRequest{
		orig:  internal.Alloc[internal.ExportLogsServiceRequest](st),
		state: st,
	}
}

// NewExportRequestFromLogs returns a ExportRequest from plog.Logs.
// Because ExportRequest is a wrapper for plog.Logs,
// any changes to the provided Logs struct will be reflected in the ExportRequest and vice versa.
func NewExportRequestFromLogs(ld plog.Logs) ExportRequest {
	return ExportRequest{
		orig:  internal.GetLogsOrig(internal.LogsWrapper(ld)),
		state: internal.GetLogsState(internal.LogsWrapper(ld)),
	}
}

// MarshalProto marshals ExportRequest into proto bytes.
func (ms ExportRequest) MarshalProto() ([]byte, error) {
	size := ms.orig.SizeProto()
	buf := make([]byte, size)
	_ = ms.orig.MarshalProto(buf)
	return buf, nil
}

// UnmarshalProto unmarshalls ExportRequest from proto bytes.
func (ms ExportRequest) UnmarshalProto(data []byte) error {
	ms.state.RetainWire(data)
	err := ms.orig.UnmarshalProtoState(data, ms.state)
	if err != nil {
		return err
	}
	otlp.MigrateLogs(ms.orig.ResourceLogs)
	return nil
}

// MarshalJSON marshals ExportRequest into JSON bytes.
func (ms ExportRequest) MarshalJSON() ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	ms.orig.MarshalJSON(dest)
	if dest.Error() != nil {
		return nil, dest.Error()
	}
	return slices.Clone(dest.Buffer()), nil
}

// UnmarshalJSON unmarshalls ExportRequest from JSON bytes.
func (ms ExportRequest) UnmarshalJSON(data []byte) error {
	iter := json.BorrowIterator(data)
	defer json.ReturnIterator(iter)
	ms.orig.UnmarshalJSONState(iter, ms.state)
	return iter.Error()
}

func (ms ExportRequest) Logs() plog.Logs {
	return plog.Logs(internal.NewLogsWrapper(ms.orig, ms.state))
}
