// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	otlperrors "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const dataFormatProtobuf = "protobuf"

// Receiver is the type used to handle spans from OpenTelemetry exporters.
type Receiver struct {
	ptraceotlp.UnimplementedGRPCServer
	nextConsumer consumer.Traces
	obsreport    *receiverhelper.ObsReport
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Traces, obsreport *receiverhelper.ObsReport) *Receiver {
	return &Receiver{
		nextConsumer: nextConsumer,
		obsreport:    obsreport,
	}
}

// Export implements the service Export traces func.
func (r *Receiver) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	td := req.Traces()
	// We need to ensure that it propagates the receiver name as a tag
	numSpans := td.SpanCount()
	if numSpans == 0 {
		return ptraceotlp.NewExportResponse(), nil
	}

	ctx = r.obsreport.StartTracesOp(ctx)
	err := r.nextConsumer.ConsumeTraces(ctx, td)
	r.obsreport.EndTracesOp(ctx, dataFormatProtobuf, numSpans, err)

	// Use appropriate status codes for permanent/non-permanent errors
	// If we return the error straightaway, then the grpc implementation will set status code to Unknown
	// Refer: https://github.com/grpc/grpc-go/blob/v1.59.0/server.go#L1345
	// So, convert the error to appropriate grpc status and return the error
	// NonPermanent errors will be converted to codes.Unavailable (equivalent to HTTP 503)
	// Permanent errors will be converted to codes.InvalidArgument (equivalent to HTTP 400)
	if err != nil {
		// See if an error is retryable.
		cet := consumererror.Traces{}
		if errors.As(err, &cet) {
			retries := 3

			// Retry three times; the real logic would be more complex akin to consumerretry.
			for i := 0; i < retries; i++ {
				// Re-use the data that was given to be retried.
				td = cet.Data()

				// Update the context to include a key that indicates the component to be retried.
				retryCtx := context.WithValue(ctx, consumererror.ComponentIDKey, cet.ID())

				// Retry the data.
				err = r.nextConsumer.ConsumeTraces(retryCtx, td)

				if err == nil {
					return ptraceotlp.NewExportResponse(), nil
				}
			}

			return ptraceotlp.NewExportResponse(), otlperrors.GetStatusFromError(err)
		}

	}

	return ptraceotlp.NewExportResponse(), nil
}
