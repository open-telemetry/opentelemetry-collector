// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/ptrace"
	reqint "go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

// MarshalTraces marshals ptrace.Traces along with the context into a byte slice.
func MarshalTraces(ctx context.Context, ld ptrace.Traces) ([]byte, error) {
	otlpTraces := internal.TracesToProto(internal.Traces(ld))
	rc := encodeContext(ctx)
	tr := reqint.TracesRequest{
		FormatVersion:  requestFormatVersion,
		TracesData:     &otlpTraces,
		RequestContext: &rc,
	}
	return proto.Marshal(&tr)
}

// UnmarshalTraces unmarshals a byte slice into ptrace.Traces and the context.
func UnmarshalTraces(buf []byte) (context.Context, ptrace.Traces, error) {
	if !isRequestPayloadV1(buf) {
		return context.Background(), ptrace.Traces{}, ErrInvalidFormat
	}
	tr := reqint.TracesRequest{}
	if err := proto.Unmarshal(buf, &tr); err != nil {
		return context.Background(), ptrace.Traces{}, fmt.Errorf("failed to unmarshal traces request: %w", err)
	}
	return decodeContext(tr.RequestContext), ptrace.Traces(internal.TracesFromProto(*tr.TracesData)), nil
}
