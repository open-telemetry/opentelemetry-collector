// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var tracesMarshaler = ptrace.NewProtoMarshaler()
var tracesUnmarshaler = ptrace.NewProtoUnmarshaler()

type tracesRequest struct {
	baseRequest
	td     ptrace.Traces
	pusher consumer.ConsumeTracesFunc
}

func newTracesRequest(ctx context.Context, td ptrace.Traces, pusher consumer.ConsumeTracesFunc) request {
	return &tracesRequest{
		baseRequest: baseRequest{ctx: ctx},
		td:          td,
		pusher:      pusher,
	}
}

func newTraceRequestUnmarshalerFunc(pusher consumer.ConsumeTracesFunc) internal.RequestUnmarshaler {
	return func(bytes []byte) (internal.PersistentRequest, error) {
		traces, err := tracesUnmarshaler.UnmarshalTraces(bytes)
		if err != nil {
			return nil, err
		}
		return newTracesRequest(context.Background(), traces, pusher), nil
	}
}

// Marshal provides serialization capabilities required by persistent queue
func (req *tracesRequest) Marshal() ([]byte, error) {
	return tracesMarshaler.MarshalTraces(req.td)
}

func (req *tracesRequest) onError(err error) request {
	var traceError consumererror.Traces
	if errors.As(err, &traceError) {
		return newTracesRequest(req.ctx, traceError.GetTraces(), req.pusher)
	}
	return req
}

func (req *tracesRequest) export(ctx context.Context) error {
	return req.pusher(ctx, req.td)
}

func (req *tracesRequest) count() int {
	return req.td.SpanCount()
}

type traceExporter struct {
	*baseExporter
	consumer.Traces
}

// NewTracesExporter creates a TracesExporter that records observability metrics and wraps every request with a Span.
func NewTracesExporter(
	cfg config.Exporter,
	set component.ExporterCreateSettings,
	pusher consumer.ConsumeTracesFunc,
	options ...Option,
) (component.TracesExporter, error) {

	if cfg == nil {
		return nil, errNilConfig
	}

	if set.Logger == nil {
		return nil, errNilLogger
	}

	if pusher == nil {
		return nil, errNilPushTraceData
	}

	bs := fromOptions(options...)
	be := newBaseExporter(cfg, set, bs, config.TracesDataType, newTraceRequestUnmarshalerFunc(pusher))
	be.wrapConsumerSender(func(nextSender requestSender) requestSender {
		return &tracesExporterWithObservability{
			obsrep:     be.obsrep,
			nextSender: nextSender,
		}
	})

	tc, err := consumer.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		req := newTracesRequest(ctx, td, pusher)
		err := be.sender.send(req)
		if errors.Is(err, errSendingQueueIsFull) {
			be.obsrep.recordTracesEnqueueFailure(req.context(), int64(req.count()))
		}
		return err
	}, bs.consumerOptions...)

	return &traceExporter{
		baseExporter: be,
		Traces:       tc,
	}, err
}

type tracesExporterWithObservability struct {
	obsrep     *obsExporter
	nextSender requestSender
}

func (tewo *tracesExporterWithObservability) send(req request) error {
	req.setContext(tewo.obsrep.StartTracesOp(req.context()))
	// Forward the data to the next consumer (this pusher is the next).
	err := tewo.nextSender.send(req)
	tewo.obsrep.EndTracesOp(req.context(), req.count(), err)
	return err
}
