// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entities // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/entities"

import (
	"context"

	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pentity/pentityotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const dataFormatProtobuf = "protobuf"

// Receiver is the type used to handle entities from OpenTelemetry exporters.
type Receiver struct {
	pentityotlp.UnimplementedGRPCServer
	nextConsumer xconsumer.Entities
	obsreport    *receiverhelper.ObsReport
}

// New creates a new Receiver reference.
func New(nextConsumer xconsumer.Entities, obsreport *receiverhelper.ObsReport) *Receiver {
	return &Receiver{
		nextConsumer: nextConsumer,
		obsreport:    obsreport,
	}
}

// Export implements the service Export entities func.
func (r *Receiver) Export(ctx context.Context, req pentityotlp.ExportRequest) (pentityotlp.ExportResponse, error) {
	ld := req.Entities()
	numSpans := ld.EntityCount()
	if numSpans == 0 {
		return pentityotlp.NewExportResponse(), nil
	}

	ctx = r.obsreport.StartEntitiesOp(ctx)
	err := r.nextConsumer.ConsumeEntities(ctx, ld)
	r.obsreport.EndEntitiesOp(ctx, dataFormatProtobuf, numSpans, err)

	// Use appropriate status codes for permanent/non-permanent errors
	// If we return the error straightaway, then the grpc implementation will set status code to Unknown
	// Refer: https://github.com/grpc/grpc-go/blob/v1.59.0/server.go#L1345
	// So, convert the error to appropriate grpc status and return the error
	// NonPermanent errors will be converted to codes.Unavailable (equivalent to HTTP 503)
	// Permanent errors will be converted to codes.InvalidArgument (equivalent to HTTP 400)
	if err != nil {
		return pentityotlp.NewExportResponse(), errors.GetStatusFromError(err)
	}

	return pentityotlp.NewExportResponse(), nil
}
