// Copyright 2018, OpenCensus Authors
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

	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/observability"
)

const (
	defaultNumWorkers = 4

	messageChannelSize = 64
)

// Receiver is the type used to handle spans from OpenCensus exporters.
type Receiver struct {
	nextConsumer consumer.TraceConsumer
	numWorkers   int
	workers      []*receiverWorker
	messageChan  chan *traceDataWithCtx
}

type traceDataWithCtx struct {
	data *data.TraceData
	ctx  context.Context
}

// New creates a new opencensus.Receiver reference.
func New(nextConsumer consumer.TraceConsumer, opts ...Option) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, errors.New("needs a non-nil consumer.TraceConsumer")
	}

	messageChan := make(chan *traceDataWithCtx, messageChannelSize)
	ocr := &Receiver{
		nextConsumer: nextConsumer,
		numWorkers:   defaultNumWorkers,
		messageChan:  messageChan,
	}
	for _, opt := range opts {
		opt(ocr)
	}

	// Setup and startup worker pool
	workers := make([]*receiverWorker, 0, ocr.numWorkers)
	for index := 0; index < ocr.numWorkers; index++ {
		worker := newReceiverWorker(ocr)
		go worker.listenOn(messageChan)
		workers = append(workers, worker)
	}
	ocr.workers = workers

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
		// If a Node has been sent from downstream, save and use it.
		if recv.Node != nil {
			lastNonNilNode = recv.Node
		}

		// TODO(songya): differentiate between unset and nil resource. See
		// https://github.com/census-instrumentation/opencensus-proto/issues/146.
		if recv.Resource != nil {
			resource = recv.Resource
		}

		td := &data.TraceData{
			Node:         lastNonNilNode,
			Resource:     resource,
			Spans:        recv.Spans,
			SourceFormat: "oc_trace",
		}

		ocr.messageChan <- &traceDataWithCtx{data: td, ctx: ctxWithReceiverName}

		observability.RecordTraceReceiverMetrics(ctxWithReceiverName, len(td.Spans), 0)

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

// Stop the receiver and its workers
func (ocr *Receiver) Stop() {
	for _, worker := range ocr.workers {
		worker.stopListening()
	}
}

type receiverWorker struct {
	receiver *Receiver
	tes      agenttracepb.TraceService_ExportServer
	cancel   chan struct{}
}

func newReceiverWorker(receiver *Receiver) *receiverWorker {
	return &receiverWorker{
		receiver: receiver,
		cancel:   make(chan struct{}),
	}
}

func (rw *receiverWorker) listenOn(cn <-chan *traceDataWithCtx) {
	for {
		select {
		case tdWithCtx := <-cn:
			rw.export(tdWithCtx.ctx, tdWithCtx.data)
		case <-rw.cancel:
			return
		}
	}
}

func (rw *receiverWorker) stopListening() {
	close(rw.cancel)
}

func (rw *receiverWorker) export(longLivedCtx context.Context, tracedata *data.TraceData) {
	if tracedata == nil {
		return
	}

	if len(tracedata.Spans) == 0 {
		return
	}

	// Trace this method
	ctx, span := trace.StartSpan(context.Background(), "OpenCensusTraceReceiver.Export")
	defer span.End()

	// TODO: (@odeke-em) investigate if it is necessary
	// to group nodes with their respective spans during
	// spansAndNode list unfurling then send spans grouped per node

	// If the starting RPC has a parent span, then add it as a parent link.
	observability.SetParentLink(longLivedCtx, span)

	rw.receiver.nextConsumer.ConsumeTraceData(ctx, *tracedata)

	span.Annotate([]trace.Attribute{
		trace.Int64Attribute("num_spans", int64(len(tracedata.Spans))),
	}, "")
}
