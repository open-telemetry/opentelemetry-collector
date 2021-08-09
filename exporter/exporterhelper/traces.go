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

package exporterhelper

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumerhelper"
	"go.opentelemetry.io/collector/model/pdata"
)

type tracesRequest struct {
	baseRequest
	td     pdata.Traces
	pusher consumerhelper.ConsumeTracesFunc
}

func newTracesRequest(ctx context.Context, td pdata.Traces, pusher consumerhelper.ConsumeTracesFunc) request {
	return &tracesRequest{
		baseRequest: baseRequest{ctx: ctx},
		td:          td,
		pusher:      pusher,
	}
}

func (req *tracesRequest) onError(err error) request {
	var traceError consumererror.Traces
	if consumererror.AsTraces(err, &traceError) {
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
	pusher consumerhelper.ConsumeTracesFunc,
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
	be := newBaseExporter(cfg, set, bs)
	be.wrapConsumerSender(func(nextSender requestSender) requestSender {
		return &tracesExporterWithObservability{
			obsrep:     be.obsrep,
			nextSender: nextSender,
		}
	})

	tc, err := consumerhelper.NewTraces(func(ctx context.Context, td pdata.Traces) error {
		req := newTracesRequest(ctx, td, pusher)
		err := be.sender.send(req)
		if errors.Is(err, errSendingQueueIsFull) {
			be.obsrep.recordTracesEnqueueFailure(req.context(), req.count())
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
