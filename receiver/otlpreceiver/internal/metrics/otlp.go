// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const dataFormatProtobuf = "protobuf"

// Receiver is the type used to handle metrics from OpenTelemetry exporters.
type Receiver struct {
	pmetricotlp.UnimplementedGRPCServer
	nextConsumer consumer.Metrics
	obsreport    *receiverhelper.ObsReport

	itemsLimiter extensionlimiter.ResourceLimiter
	sizeLimiter  extensionlimiter.ResourceLimiter
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Metrics, obsreport *receiverhelper.ObsReport, limiter extensionlimiter.Provider) *Receiver {
	// TODO@@@: Move this limiter logic to the receiverhelper package.
	var itemsLimiter, sizeLimiter extensionlimiter.ResourceLimiter
	if limiter != nil {
		itemsLimiter = limiter.ResourceLimiter(extensionlimiter.WeightKeyRequestItems)
		sizeLimiter = limiter.ResourceLimiter(extensionlimiter.WeightKeyResidentSize)
	}
	return &Receiver{
		nextConsumer: nextConsumer,
		obsreport:    obsreport,
		itemsLimiter: itemsLimiter,
		sizeLimiter:  sizeLimiter,
	}
}

// Export implements the service Export metrics func.
func (r *Receiver) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	md := req.Metrics()
	dataPointCount := md.DataPointCount()
	if dataPointCount == 0 {
		return pmetricotlp.NewExportResponse(), nil
	}

	// Acquire items limiter if available
	if r.itemsLimiter != nil {
		release, err := r.itemsLimiter.Acquire(ctx, uint64(dataPointCount))
		if err != nil {
			return pmetricotlp.NewExportResponse(), errors.GetStatusFromError(err)
		}
		defer release()
	}

	// Acquire size limiter if available
	if r.sizeLimiter != nil {
		var sizer pmetric.ProtoMarshaler
		size := sizer.MetricsSize(md)
		release, err := r.sizeLimiter.Acquire(ctx, uint64(size))
		if err != nil {
			return pmetricotlp.NewExportResponse(), errors.GetStatusFromError(err)
		}
		defer release()
	}

	ctx = r.obsreport.StartMetricsOp(ctx)
	err := r.nextConsumer.ConsumeMetrics(ctx, md)
	r.obsreport.EndMetricsOp(ctx, dataFormatProtobuf, dataPointCount, err)

	// Use appropriate status codes for permanent/non-permanent errors
	// If we return the error straightaway, then the grpc implementation will set status code to Unknown
	// Refer: https://github.com/grpc/grpc-go/blob/v1.59.0/server.go#L1345
	// So, convert the error to appropriate grpc status and return the error
	// NonPermanent errors will be converted to codes.Unavailable (equivalent to HTTP 503)
	// Permanent errors will be converted to codes.InvalidArgument (equivalent to HTTP 400)
	if err != nil {
		return pmetricotlp.NewExportResponse(), errors.GetStatusFromError(err)
	}

	return pmetricotlp.NewExportResponse(), nil
}
