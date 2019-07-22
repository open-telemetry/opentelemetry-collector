// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package octrace

import (
	"context"
	"errors"
	"io"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/observability"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

const (
	defaultNumWorkers = 4

	messageChannelSize = 64
)

// Receiver is the type used to handle spans from OpenCensus exporters.
type Receiver struct {
	nextConsumer        consumer.TraceConsumer
	host                receiver.Host
	backPressureSetting configmodels.BackPressureSetting
}

type traceDataWithCtx struct {
	data *consumerdata.TraceData
	ctx  context.Context
}

// New creates a new opencensus.Receiver reference.
func New(nextConsumer consumer.TraceConsumer, host receiver.Host, opts ...Option) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, errors.New("nil nextConsumer")
	}
	if host == nil {
		return nil, errors.New("nil host")
	}

	ocr := &Receiver{
		nextConsumer: nextConsumer,
		host:         host,
	}
	for _, opt := range opts {
		opt(ocr)
	}

	return ocr, nil
}

var _ agenttracepb.TraceServiceServer = (*Receiver)(nil)

var errUnimplemented = errors.New("unimplemented")

// Config handles configuration messages.
func (ocr *Receiver) Config(tcs agenttracepb.TraceService_ConfigServer) error {
	// TODO: Implement when we define the config receiver/sender.
	return errUnimplemented
}

var errTraceExportProtocolViolation = errors.New("protocol violation: Export's first message must have a Node")

const receiverTagValue = "oc_trace"

// Export is the gRPC method that receives streamed traces from
// OpenCensus-traceproto compatible libraries/applications.
func (ocr *Receiver) Export(tes agenttracepb.TraceService_ExportServer) error {
	// We need to ensure that it propagates the receiver name as a tag
	ctxWithReceiverName := observability.ContextWithReceiverName(tes.Context(), receiverTagValue)

	// The first message MUST have a non-nil Node.
	recv, err := tes.Recv()
	if err != nil {
		return err
	}

	// Check the condition that the first message has a non-nil Node.
	if recv.Node == nil {
		return errTraceExportProtocolViolation
	}

	var lastNonNilNode *commonpb.Node
	var resource *resourcepb.Resource
	// Now that we've got the first message with a Node, we can start to receive streamed up spans.
	for {
		lastNonNilNode, resource, err = ocr.processReceivedMsg(ctxWithReceiverName, lastNonNilNode, resource, recv)
		if err != nil {
			return err
		}

		recv, err = tes.Recv()
		if err != nil {
			if err == io.EOF {
				// Do not return EOF as an error so that grpc-gateway calls get an empty
				// response with HTTP status code 200 rather than a 500 error with EOF.
				return nil
			}
			return err
		}
	}
}

func (ocr *Receiver) processReceivedMsg(
	receiverCtx context.Context,
	lastNonNilNode *commonpb.Node,
	resource *resourcepb.Resource,
	recv *agenttracepb.ExportTraceServiceRequest,
) (*commonpb.Node, *resourcepb.Resource, error) {
	// If a Node has been sent from downstream, save and use it.
	if recv.Node != nil {
		lastNonNilNode = recv.Node
	}

	// TODO(songya): differentiate between unset and nil resource. See
	// https://github.com/census-instrumentation/opencensus-proto/issues/146.
	if recv.Resource != nil {
		resource = recv.Resource
	}

	td := &consumerdata.TraceData{
		Node:         lastNonNilNode,
		Resource:     resource,
		Spans:        recv.Spans,
		SourceFormat: "oc_trace",
	}

	return lastNonNilNode, resource, ocr.export(receiverCtx, td)
}

func (ocr *Receiver) export(receiverCtx context.Context, tracedata *consumerdata.TraceData) error {
	// Trace this method
	ctx, span := trace.StartSpan(context.Background(), "OpenCensusTraceReceiver.Export")
	defer span.End()

	// If the starting RPC has a parent span, then add it as a parent link.
	observability.SetParentLink(receiverCtx, span)

	if !ocr.host.OkToIngest() {
		statusCode, zPageMessage := observability.RecordIngestionBlocked(
			receiverCtx,
			span,
			ocr.backPressureSetting)

		return status.Error(codes.Code(statusCode), zPageMessage)
	}

	receivedSpans := len(tracedata.Spans)
	droppedSpans := 0
	err := ocr.nextConsumer.ConsumeTraceData(ctx, *tracedata)
	if err != nil {
		droppedSpans = receivedSpans
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeUnknown,
			Message: err.Error(),
		})
	}

	observability.RecordTraceReceiverMetrics(receiverCtx, receivedSpans, droppedSpans)
	return err
}
