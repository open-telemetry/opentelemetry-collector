// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"

import (
	"context"
	stderrors "errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	dataFormatProtobuf = "protobuf"
	invalidUTF8Message = "request contains invalid UTF-8"
)

// errInvalidUTF8 is reported to obsreport so that data rejected for invalid UTF-8
// is accounted for in the receiver metrics and spans.
var errInvalidUTF8 = consumererror.NewPermanent(stderrors.New(invalidUTF8Message))

// Receiver is the type used to handle metrics from OpenTelemetry exporters.
type Receiver struct {
	pmetricotlp.UnimplementedGRPCServer
	nextConsumer consumer.Metrics
	obsreport    *receiverhelper.ObsReport
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Metrics, obsreport *receiverhelper.ObsReport) *Receiver {
	return &Receiver{
		nextConsumer: nextConsumer,
		obsreport:    obsreport,
	}
}

// Export implements the service Export metrics func.
func (r *Receiver) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	rejectedDataPoints := 0
	if metadata.ReceiverOtlpRejectInvalidUTF8FeatureGate.IsEnabled() {
		rejectedDataPoints = req.RejectInvalidUTF8()
	}
	resp := pmetricotlp.NewExportResponse()
	if rejectedDataPoints > 0 {
		resp.PartialSuccess().SetRejectedDataPoints(int64(rejectedDataPoints))
		resp.PartialSuccess().SetErrorMessage(invalidUTF8Message)
		r.recordRejected(ctx, rejectedDataPoints)
	}
	md := req.Metrics()
	dataPointCount := md.DataPointCount()
	if dataPointCount == 0 {
		return resp, nil
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

	return resp, nil
}

// recordRejected accounts for data points rejected by the receiver itself so they
// show up in the receiver telemetry (e.g. otelcol_receiver_refused_metric_points) and
// not only in the partial-success response.
func (r *Receiver) recordRejected(ctx context.Context, rejected int) {
	ctx = r.obsreport.StartMetricsOp(ctx)
	r.obsreport.EndMetricsOp(ctx, dataFormatProtobuf, rejected, errInvalidUTF8)
}
