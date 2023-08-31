// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const dataFormatProtobuf = "protobuf"

// Receiver is the type used to handle metrics from OpenTelemetry exporters.
type Receiver struct {
	pmetricotlp.UnimplementedGRPCServer
	nextConsumer consumer.Metrics
	obsrecv      *obsreport.Receiver
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Metrics, obsrecv *obsreport.Receiver) *Receiver {
	return &Receiver{
		nextConsumer: nextConsumer,
		obsrecv:      obsrecv,
	}
}

// Export implements the service Export metrics func.
func (r *Receiver) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	md := req.Metrics()
	dataPointCount := md.DataPointCount()
	if dataPointCount == 0 {
		return pmetricotlp.NewExportResponse(), nil
	}

	ctx = r.obsrecv.StartMetricsOp(ctx)
	err := r.nextConsumer.ConsumeMetrics(ctx, md)
	r.obsrecv.EndMetricsOp(ctx, dataFormatProtobuf, dataPointCount, err)
	// Use appropiate status codes for permanent/non-permanent errors
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			code := codes.Unavailable
			if consumererror.IsPermanent(err) {
				code = codes.InvalidArgument
			}
			s = status.New(code, err.Error())
		}
		return pmetricotlp.NewExportResponse(), s.Err()
	}

	return pmetricotlp.NewExportResponse(), nil
}
