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

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// PushLogs is a helper function that is similar to ConsumeLogs but also returns
// the number of dropped logs.
type PushLogs func(ctx context.Context, md pdata.Logs) (droppedTimeSeries int, err error)

type logsRequest struct {
	baseRequest
	ld     pdata.Logs
	pusher PushLogs
}

func newLogsRequest(ctx context.Context, ld pdata.Logs, pusher PushLogs) request {
	return &logsRequest{
		baseRequest: baseRequest{ctx: ctx},
		ld:          ld,
		pusher:      pusher,
	}
}

func (req *logsRequest) onPartialError(partialErr consumererror.PartialError) request {
	return newLogsRequest(req.ctx, partialErr.GetLogs(), req.pusher)
}

func (req *logsRequest) export(ctx context.Context) (int, error) {
	return req.pusher(ctx, req.ld)
}

func (req *logsRequest) count() int {
	return req.ld.LogRecordCount()
}

type logsExporter struct {
	*baseExporter
	pusher PushLogs
}

func (lexp *logsExporter) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	exporterCtx := obsreport.ExporterContext(ctx, lexp.cfg.Name())
	_, err := lexp.sender.send(newLogsRequest(exporterCtx, ld, lexp.pusher))
	return err
}

// NewLogsExporter creates an LogsExporter that records observability metrics and wraps every request with a Span.
func NewLogsExporter(
	cfg configmodels.Exporter,
	logger *zap.Logger,
	pusher PushLogs,
	options ...Option,
) (component.LogsExporter, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	if logger == nil {
		return nil, errNilLogger
	}

	if pusher == nil {
		return nil, errNilPushLogsData
	}

	be := newBaseExporter(cfg, logger, options...)
	be.wrapConsumerSender(func(nextSender requestSender) requestSender {
		return &logsExporterWithObservability{
			obsrep:     obsreport.NewExporterObsReport(configtelemetry.GetMetricsLevelFlagValue(), cfg.Name()),
			nextSender: nextSender,
		}
	})

	return &logsExporter{
		baseExporter: be,
		pusher:       pusher,
	}, nil
}

type logsExporterWithObservability struct {
	obsrep     *obsreport.ExporterObsReport
	nextSender requestSender
}

func (lewo *logsExporterWithObservability) send(req request) (int, error) {
	req.setContext(lewo.obsrep.StartLogsExportOp(req.context()))
	numDroppedLogs, err := lewo.nextSender.send(req)
	lewo.obsrep.EndLogsExportOp(req.context(), req.count(), err)
	return numDroppedLogs, err
}
