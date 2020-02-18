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

	"github.com/open-telemetry/opentelemetry-collector/client"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
)

const (
	receiverTagValue  = "oc_trace"
	receiverTransport = "grpc" // TODO: transport is being hard coded for now, investigate if info is available on context.
)

// Receiver is the type used to handle spans from OpenCensus exporters.
type Receiver struct {
	nextConsumer consumer.TraceConsumer
	instanceName string
}

// New creates a new opencensus.Receiver reference.
func New(instanceName string, nextConsumer consumer.TraceConsumer, opts ...Option) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, oterr.ErrNilNextConsumer
	}

	ocr := &Receiver{
		nextConsumer: nextConsumer,
		instanceName: instanceName,
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

// Export is the gRPC method that receives streamed traces from
// OpenCensus-traceproto compatible libraries/applications.
func (ocr *Receiver) Export(tes agenttracepb.TraceService_ExportServer) error {
	// We need to ensure that it propagates the receiver name as a tag
	ctxWithReceiverName := obsreport.ReceiverContext(tes.Context(), ocr.instanceName, receiverTransport, receiverTagValue)

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
			// Metrics and z-pages record data loss but there is no back pressure.
			// However, cause the stream to be closed.
			return nil
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
	longLivedRPCCtx context.Context,
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

	err := ocr.sendToNextConsumer(longLivedRPCCtx, td)
	return lastNonNilNode, resource, err
}

func (ocr *Receiver) sendToNextConsumer(longLivedRPCCtx context.Context, tracedata *consumerdata.TraceData) error {
	// Do not use longLivedRPCCtx to start the span so this trace ends right at this
	// function, and the span is not a child of any span from the stream context.
	ctx, span := obsreport.StartTraceDataReceiveOp(
		context.Background(),
		ocr.instanceName,
		receiverTransport)

	// If the starting RPC has a parent span, then add it as a parent link.
	observability.SetParentLink(longLivedRPCCtx, span)

	if c, ok := client.FromGRPC(longLivedRPCCtx); ok {
		ctx = client.NewContext(ctx, c)
	}

	var err error
	numSpans := 0
	if tracedata != nil && len(tracedata.Spans) != 0 {
		numSpans = len(tracedata.Spans)
		err = ocr.nextConsumer.ConsumeTraceData(ctx, *tracedata)
	}

	obsreport.EndTraceDataReceiveOp(longLivedRPCCtx, span, "protobuf", numSpans, err)

	return err
}
