// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/plog"
	reqint "go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

// MarshalLogs marshals plog.Logs along with the context into a byte slice.
func MarshalLogs(ctx context.Context, ld plog.Logs) ([]byte, error) {
	otlpLogs := internal.LogsToProto(internal.Logs(ld))
	rc := encodeContext(ctx)
	lr := reqint.LogsRequest{
		FormatVersion:  requestFormatVersion,
		LogsData:       &otlpLogs,
		RequestContext: &rc,
	}
	return lr.Marshal()
}

// UnmarshalLogs unmarshals a byte slice into plog.Logs and the context.
func UnmarshalLogs(buf []byte) (context.Context, plog.Logs, error) {
	ctx := context.Background()
	if !isRequestPayloadV1(buf) {
		return ctx, plog.Logs{}, ErrInvalidFormat
	}
	lr := reqint.LogsRequest{}
	if err := lr.Unmarshal(buf); err != nil {
		return ctx, plog.Logs{}, fmt.Errorf("failed to unmarshal logs request: %w", err)
	}
	return decodeContext(ctx, lr.RequestContext), plog.Logs(internal.LogsFromProto(*lr.LogsData)), nil
}
