// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MarshalMetrics marshals pmetric.Metrics along with the context into a byte slice.
func MarshalMetrics(ctx context.Context, ld pmetric.Metrics) ([]byte, error) {
	mr := internal.MetricsRequest{
		FormatVersion:  requestFormatVersion,
		MetricsData:    internal.MetricsToProto(internal.MetricsWrapper(ld)),
		RequestContext: encodeContext(ctx),
	}
	buf := make([]byte, mr.SizeProto())
	mr.MarshalProto(buf)
	return buf, nil
}

// UnmarshalMetrics unmarshals a byte slice into pmetric.Metrics and the context.
func UnmarshalMetrics(buf []byte) (context.Context, pmetric.Metrics, error) {
	ctx := context.Background()
	if !isRequestPayloadV1(buf) {
		return ctx, pmetric.Metrics{}, ErrInvalidFormat
	}
	mr := internal.MetricsRequest{}
	if err := mr.UnmarshalProto(buf); err != nil {
		return ctx, pmetric.Metrics{}, fmt.Errorf("failed to unmarshal metrics request: %w", err)
	}
	return decodeContext(ctx, mr.RequestContext), pmetric.Metrics(internal.MetricsFromProto(mr.MetricsData)), nil
}
