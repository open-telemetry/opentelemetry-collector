// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logs // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"

import (
	"context"
	stderrors "errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
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

// Receiver is the type used to handle logs from OpenTelemetry exporters.
type Receiver struct {
	plogotlp.UnimplementedGRPCServer
	nextConsumer consumer.Logs
	obsreport    *receiverhelper.ObsReport
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Logs, obsreport *receiverhelper.ObsReport) *Receiver {
	return &Receiver{
		nextConsumer: nextConsumer,
		obsreport:    obsreport,
	}
}

// Export implements the service Export logs func.
func (r *Receiver) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	rejectedLogRecords := 0
	if metadata.ReceiverOtlpRejectInvalidUTF8FeatureGate.IsEnabled() {
		rejectedLogRecords = req.RejectInvalidUTF8()
	}
	resp := plogotlp.NewExportResponse()
	if rejectedLogRecords > 0 {
		resp.PartialSuccess().SetRejectedLogRecords(int64(rejectedLogRecords))
		resp.PartialSuccess().SetErrorMessage(invalidUTF8Message)
		r.recordRejected(ctx, rejectedLogRecords)
	}
	ld := req.Logs()
	numSpans := ld.LogRecordCount()
	if numSpans == 0 {
		return resp, nil
	}

	ctx = r.obsreport.StartLogsOp(ctx)
	err := r.nextConsumer.ConsumeLogs(ctx, ld)
	r.obsreport.EndLogsOp(ctx, dataFormatProtobuf, numSpans, err)

	// Use appropriate status codes for permanent/non-permanent errors
	// If we return the error straightaway, then the grpc implementation will set status code to Unknown
	// Refer: https://github.com/grpc/grpc-go/blob/v1.59.0/server.go#L1345
	// So, convert the error to appropriate grpc status and return the error
	// NonPermanent errors will be converted to codes.Unavailable (equivalent to HTTP 503)
	// Permanent errors will be converted to codes.InvalidArgument (equivalent to HTTP 400)
	if err != nil {
		return plogotlp.NewExportResponse(), errors.GetStatusFromError(err)
	}

	return resp, nil
}

// recordRejected accounts for log records rejected by the receiver itself so they
// show up in the receiver telemetry (e.g. otelcol_receiver_refused_log_records) and
// not only in the partial-success response.
func (r *Receiver) recordRejected(ctx context.Context, rejected int) {
	ctx = r.obsreport.StartLogsOp(ctx)
	r.obsreport.EndLogsOp(ctx, dataFormatProtobuf, rejected, errInvalidUTF8)
}
