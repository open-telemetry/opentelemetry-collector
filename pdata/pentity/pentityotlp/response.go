// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentityotlp // import "go.opentelemetry.io/collector/pdata/pentity/pentityotlp"

import (
	"bytes"

	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorentity "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/entities/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

// ExportResponse represents the response for gRPC/HTTP client/server.
type ExportResponse struct {
	orig  *otlpcollectorentity.ExportEntitiesServiceResponse
	state *internal.State
}

// NewExportResponse returns an empty ExportResponse.
func NewExportResponse() ExportResponse {
	state := internal.StateMutable
	return ExportResponse{
		orig:  &otlpcollectorentity.ExportEntitiesServiceResponse{},
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
	var buf bytes.Buffer
	if err := json.Marshal(&buf, ms.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls ExportResponse from JSON bytes.
func (ms ExportResponse) UnmarshalJSON(data []byte) error {
	iter := jsoniter.ConfigFastest.BorrowIterator(data)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	ms.unmarshalJsoniter(iter)
	return iter.Error
}

// PartialSuccess returns the ExportPartialSuccess associated with this ExportResponse.
func (ms ExportResponse) PartialSuccess() ExportPartialSuccess {
	return newExportPartialSuccess(&ms.orig.PartialSuccess, ms.state)
}

func (ms ExportResponse) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "partial_success", "partialSuccess":
			ms.PartialSuccess().unmarshalJsoniter(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ExportPartialSuccess) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(_ *jsoniter.Iterator, f string) bool {
		switch f {
		case "rejected_entities", "rejectedEntities":
			ms.orig.RejectedEntities = json.ReadInt64(iter)
		case "error_message", "errorMessage":
			ms.orig.ErrorMessage = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}
