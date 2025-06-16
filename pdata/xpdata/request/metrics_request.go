// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pmetric"
	reqint "go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

// MarshalMetrics marshals pmetric.Metrics along with the context into a byte slice.
func MarshalMetrics(ctx context.Context, ld pmetric.Metrics) ([]byte, error) {
	otlpMetrics := internal.MetricsToProto(internal.Metrics(ld))
	rc := encodeContext(ctx)
	mr := reqint.MetricsRequest{
		FormatVersion:  requestFormatVersion,
		MetricsData:    &otlpMetrics,
		RequestContext: &rc,
	}
	return proto.Marshal(&mr)
}

// UnmarshalMetrics unmarshals a byte slice into pmetric.Metrics and the context.
func UnmarshalMetrics(buf []byte) (context.Context, pmetric.Metrics, error) {
	if !isRequestPayloadV1(buf) {
		return context.Background(), pmetric.Metrics{}, ErrInvalidFormat
	}
	mr := reqint.MetricsRequest{}
	if err := proto.Unmarshal(buf, &mr); err != nil {
		return context.Background(), pmetric.Metrics{}, fmt.Errorf("failed to unmarshal metrics request: %w", err)
	}
	return decodeContext(mr.RequestContext), pmetric.Metrics(internal.MetricsFromProto(*mr.MetricsData)), nil
}
