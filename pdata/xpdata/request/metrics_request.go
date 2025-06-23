// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"fmt"

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
	return mr.Marshal()
}

// UnmarshalMetrics unmarshals a byte slice into pmetric.Metrics and the context.
func UnmarshalMetrics(buf []byte) (context.Context, pmetric.Metrics, error) {
	ctx := context.Background()
	if !isRequestPayloadV1(buf) {
		return ctx, pmetric.Metrics{}, ErrInvalidFormat
	}
	mr := reqint.MetricsRequest{}
	if err := mr.Unmarshal(buf); err != nil {
		return ctx, pmetric.Metrics{}, fmt.Errorf("failed to unmarshal metrics request: %w", err)
	}
	return decodeContext(ctx, mr.RequestContext), pmetric.Metrics(internal.MetricsFromProto(*mr.MetricsData)), nil
}
