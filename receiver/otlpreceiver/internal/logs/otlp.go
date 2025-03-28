// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logs // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const dataFormatProtobuf = "protobuf"

// Receiver is the type used to handle logs from OpenTelemetry exporters.
type Receiver struct {
	plogotlp.UnimplementedGRPCServer
	nextConsumer consumer.Logs
	obsreport    *receiverhelper.ObsReport
	itemsLimiter extensionlimiter.ResourceLimiter
	sizeLimiter  extensionlimiter.ResourceLimiter
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Logs, obsreport *receiverhelper.ObsReport, limiter extensionlimiter.Provider) *Receiver {
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

// Export implements the service Export logs func.
func (r *Receiver) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	ld := req.Logs()
	numRecords := ld.LogRecordCount()
	if numRecords == 0 {
		return plogotlp.NewExportResponse(), nil
	}

	// Apply the items limiter if available
	if r.itemsLimiter != nil {
		release, err := r.itemsLimiter.Acquire(ctx, uint64(numRecords))
		if err != nil {
			return plogotlp.NewExportResponse(), errors.GetStatusFromError(err)
		}
		defer release()
	}

	// Apply the memory size limiter if available
	if r.sizeLimiter != nil {
		// Get the marshaled size of the request as a proxy for memory size
		var sizer plog.ProtoMarshaler
		size := sizer.LogsSize(ld)
		release, err := r.sizeLimiter.Acquire(ctx, uint64(size))
		if err != nil {
			return plogotlp.NewExportResponse(), errors.GetStatusFromError(err)
		}
		defer release()
	}

	ctx = r.obsreport.StartLogsOp(ctx)
	err := r.nextConsumer.ConsumeLogs(ctx, ld)
	r.obsreport.EndLogsOp(ctx, dataFormatProtobuf, numRecords, err)

	// Use appropriate status codes for permanent/non-permanent errors
	// If we return the error straightaway, then the grpc implementation will set status code to Unknown
	// Refer: https://github.com/grpc/grpc-go/blob/v1.59.0/server.go#L1345
	// So, convert the error to appropriate grpc status and return the error
	// NonPermanent errors will be converted to codes.Unavailable (equivalent to HTTP 503)
	// Permanent errors will be converted to codes.InvalidArgument (equivalent to HTTP 400)
	if err != nil {
		return plogotlp.NewExportResponse(), errors.GetStatusFromError(err)
	}

	return plogotlp.NewExportResponse(), nil
}
