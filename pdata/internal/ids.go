// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"go.opentelemetry.io/collector/pdata/internal/data"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

func DeleteTraceID(*data.TraceID, bool) {}

func DeleteSpanID(*data.SpanID, bool) {}

func DeleteProfileID(*data.ProfileID, bool) {}

func GenTestTraceID() *data.TraceID {
	id := data.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	return &id
}

func GenTestSpanID() *data.SpanID {
	id := data.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
	return &id
}

func GenTestProfileID() *data.ProfileID {
	id := data.ProfileID([16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1})
	return &id
}

func MarshalJSONTraceID(id *data.TraceID, dest *json.Stream) {
	id.MarshalJSONStream(dest)
}

func MarshalJSONSpanID(id *data.SpanID, dest *json.Stream) {
	id.MarshalJSONStream(dest)
}

func MarshalJSONProfileID(id *data.ProfileID, dest *json.Stream) {
	id.MarshalJSONStream(dest)
}

func UnmarshalJSONTraceID(id *data.TraceID, iter *json.Iterator) {
	id.UnmarshalJSONIter(iter)
}

func UnmarshalJSONSpanID(id *data.SpanID, iter *json.Iterator) {
	id.UnmarshalJSONIter(iter)
}

func UnmarshalJSONProfileID(id *data.ProfileID, iter *json.Iterator) {
	id.UnmarshalJSONIter(iter)
}

func SizeProtoTraceID(id *data.TraceID) int {
	return id.Size()
}

func SizeProtoSpanID(id *data.SpanID) int {
	return id.Size()
}

func SizeProtoProfileID(id *data.ProfileID) int {
	return id.Size()
}

func MarshalProtoTraceID(id *data.TraceID, buf []byte) int {
	size := id.Size()
	_, _ = id.MarshalTo(buf[len(buf)-size:])
	return size
}

func MarshalProtoSpanID(id *data.SpanID, buf []byte) int {
	size := id.Size()
	_, _ = id.MarshalTo(buf[len(buf)-size:])
	return size
}

func MarshalProtoProfileID(id *data.ProfileID, buf []byte) int {
	size := id.Size()
	_, _ = id.MarshalTo(buf[len(buf)-size:])
	return size
}

func UnmarshalProtoTraceID(id *data.TraceID, buf []byte) error {
	return id.Unmarshal(buf)
}

func UnmarshalProtoSpanID(id *data.SpanID, buf []byte) error {
	return id.Unmarshal(buf)
}

func UnmarshalProtoProfileID(id *data.ProfileID, buf []byte) error {
	return id.Unmarshal(buf)
}
