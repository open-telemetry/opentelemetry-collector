// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

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
	return proto.Marshal(&lr)
}

// UnmarshalLogs unmarshals a byte slice into plog.Logs and the context.
func UnmarshalLogs(buf []byte) (context.Context, plog.Logs, error) {
	if !isRequestPayloadV1(buf) {
		return context.Background(), plog.Logs{}, ErrInvalidFormat
	}
	lr := reqint.LogsRequest{}
	if err := proto.Unmarshal(buf, &lr); err != nil {
		return context.Background(), plog.Logs{}, fmt.Errorf("failed to unmarshal logs request: %w", err)
	}
	return decodeContext(lr.RequestContext), plog.Logs(internal.LogsFromProto(*lr.LogsData)), nil
}
