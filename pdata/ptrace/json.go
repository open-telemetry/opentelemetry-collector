// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/ptrace/internal/ptracejson"
)

var delegate = ptracejson.JSONMarshaler

var _ Marshaler = (*JSONMarshaler)(nil)

type JSONMarshaler struct{}

func (*JSONMarshaler) MarshalTraces(td Traces) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.TracesToProto(internal.Traces(td))
	err := delegate.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

type JSONUnmarshaler struct{}

func (*JSONUnmarshaler) UnmarshalTraces(buf []byte) (Traces, error) {
	var td otlptrace.TracesData
	if err := ptracejson.UnmarshalTraceData(buf, &td); err != nil {
		return Traces{}, err
	}
	return Traces(internal.TracesFromProto(td)), nil
}
