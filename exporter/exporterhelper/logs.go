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

type logsRequest struct {
	baseRequest
	ld     pdata.Logs
	pusher consumerhelper.ConsumeLogsFunc
}

func newLogsRequest(ctx context.Context, ld pdata.Logs, pusher consumerhelper.ConsumeLogsFunc) request {
	return &logsRequest{
		baseRequest: baseRequest{ctx: ctx},
		ld:          ld,
		pusher:      pusher,
	}
}

func (req *logsRequest) onError(err error) request {
	var logError consumererror.Logs
	if consumererror.AsLogs(err, &logError) {
		return newLogsRequest(req.ctx, logError.GetLogs(), req.pusher)
	}
	return req
}

func (req *logsRequest) export(ctx context.Context) error {
	return req.pusher(ctx, req.ld)
}

func (req *logsRequest) count() int {
	return req.ld.LogRecordCount()
}

type logsExporter struct {
	*baseExporter
	consumer.Logs
}

// NewLogsExporter creates an LogsExporter that records observability metrics and wraps every request with a Span.
func NewLogsExporter(
	cfg config.Exporter,
	set component.ExporterCreateSettings,
	pusher consumerhelper.ConsumeLogsFunc,
	options ...Option,
) (component.LogsExporter, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	if set.Logger == nil {
		return nil, errNilLogger
	}

	if pusher == nil {
		return nil, errNilPushLogsData
	}

	bs := fromOptions(options...)
	be := newBaseExporter(cfg, set, bs)
	be.wrapConsumerSender(func(nextSender requestSender) requestSender {
		return &logsExporterWithObservability{
			obsrep:     be.obsrep,
			nextSender: nextSender,
		}
	})

	lc, err := consumerhelper.NewLogs(func(ctx context.Context, ld pdata.Logs) error {
		req := newLogsRequest(ctx, ld, pusher)
		err := be.sender.send(req)
		if errors.Is(err, errSendingQueueIsFull) {
			be.obsrep.recordLogsEnqueueFailure(req.context(), req.count())
		}
		return err
	}, bs.consumerOptions...)

	return &logsExporter{
		baseExporter: be,
		Logs:         lc,
	}, err
}

type logsExporterWithObservability struct {
	obsrep     *obsExporter
	nextSender requestSender
}

func (lewo *logsExporterWithObservability) send(req request) error {
	req.setContext(lewo.obsrep.StartLogsOp(req.context()))
	err := lewo.nextSender.send(req)
	lewo.obsrep.EndLogsOp(req.context(), req.count(), err)
	return err
}
