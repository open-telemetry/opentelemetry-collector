// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
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
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			code := codes.Unavailable
			if consumererror.IsPermanent(err) {
				code = codes.InvalidArgument
			}
			s = status.New(code, err.Error())
		}
		return ptraceotlp.NewExportResponse(), s.Err()
	}

	return ptraceotlp.NewExportResponse(), nil
}
