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

// PushMetrics is a helper function that is similar to ConsumeMetrics but also returns
// the number of dropped metrics.
type PushMetrics func(ctx context.Context, md pdata.Metrics) (droppedTimeSeries int, err error)

type metricsRequest struct {
	baseRequest
	md     pdata.Metrics
	pusher PushMetrics
}

func newMetricsRequest(ctx context.Context, md pdata.Metrics, pusher PushMetrics) request {
	return &metricsRequest{
		baseRequest: baseRequest{ctx: ctx},
		md:          md,
		pusher:      pusher,
	}
}

func (req *metricsRequest) onPartialError(partialErr consumererror.PartialError) request {
	return newMetricsRequest(req.ctx, partialErr.GetMetrics(), req.pusher)
}

func (req *metricsRequest) export(ctx context.Context) (int, error) {
	return req.pusher(ctx, req.md)
}

func (req *metricsRequest) count() int {
	_, numPoints := req.md.MetricAndDataPointCount()
	return numPoints
}

type metricsExporter struct {
	*baseExporter
	pusher PushMetrics
}

func (mexp *metricsExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	if mexp.baseExporter.convertResourceToTelemetry {
		md = convertResourceToLabels(md)
	}
	exporterCtx := obsreport.ExporterContext(ctx, mexp.cfg.Name())
	req := newMetricsRequest(exporterCtx, md, mexp.pusher)
	_, err := mexp.sender.send(req)
	return err
}

// NewMetricsExporter creates an MetricsExporter that records observability metrics and wraps every request with a Span.
func NewMetricsExporter(
	cfg configmodels.Exporter,
	logger *zap.Logger,
	pusher PushMetrics,
	options ...Option,
) (component.MetricsExporter, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	if logger == nil {
		return nil, errNilLogger
	}

	if pusher == nil {
		return nil, errNilPushMetricsData
	}

	be := newBaseExporter(cfg, logger, options...)
	be.wrapConsumerSender(func(nextSender requestSender) requestSender {
		return &metricsSenderWithObservability{
			obsrep:     obsreport.NewExporterObsReport(configtelemetry.GetMetricsLevelFlagValue(), cfg.Name()),
			nextSender: nextSender,
		}
	})

	return &metricsExporter{
		baseExporter: be,
		pusher:       pusher,
	}, nil
}

type metricsSenderWithObservability struct {
	obsrep     *obsreport.ExporterObsReport
	nextSender requestSender
}

func (mewo *metricsSenderWithObservability) send(req request) (int, error) {
	req.setContext(mewo.obsrep.StartMetricsExportOp(req.context()))
	_, err := mewo.nextSender.send(req)

	// TODO: this is not ideal: it should come from the next function itself.
	// 	temporarily loading it from internal format. Once full switch is done
	// 	to new metrics will remove this.
	mReq := req.(*metricsRequest)
	numReceivedMetrics, numPoints := mReq.md.MetricAndDataPointCount()

	mewo.obsrep.EndMetricsExportOp(req.context(), numPoints, err)
	return numReceivedMetrics, err
}
