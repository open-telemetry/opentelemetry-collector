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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// NumTimeSeries returns the number of timeseries in a MetricsData.
func NumTimeSeries(md consumerdata.MetricsData) int {
	receivedTimeSeries := 0
	for _, metric := range md.Metrics {
		receivedTimeSeries += len(metric.GetTimeseries())
	}
	return receivedTimeSeries
}

// PushMetricsData is a helper function that is similar to ConsumeMetricsData but also returns
// the number of dropped metrics.
type PushMetricsData func(ctx context.Context, md pdata.Metrics) (droppedTimeSeries int, err error)

type metricsRequest struct {
	baseRequest
	md     pdata.Metrics
	pusher PushMetricsData
}

func newMetricsRequest(ctx context.Context, md pdata.Metrics, pusher PushMetricsData) request {
	return &metricsRequest{
		baseRequest: baseRequest{ctx: ctx},
		md:          md,
		pusher:      pusher,
	}
}

func (req *metricsRequest) onPartialError(consumererror.PartialError) request {
	// TODO: implement this.
	return req
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
	pusher PushMetricsData
}

func (mexp *metricsExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	exporterCtx := obsreport.ExporterContext(ctx, mexp.cfg.Name())
	req := newMetricsRequest(exporterCtx, md, mexp.pusher)
	_, err := mexp.sender.send(req)
	return err
}

// NewMetricsExporter creates an MetricsExporter that records observability metrics and wraps every request with a Span.
func NewMetricsExporter(cfg configmodels.Exporter, pushMetricsData PushMetricsData, options ...ExporterOption) (component.MetricsExporter, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	if pushMetricsData == nil {
		return nil, errNilPushMetricsData
	}

	be := newBaseExporter(cfg, options...)
	be.wrapConsumerSender(func(nextSender requestSender) requestSender {
		return &metricsSenderWithObservability{
			exporterName: cfg.Name(),
			nextSender:   nextSender,
		}
	})

	return &metricsExporter{
		baseExporter: be,
		pusher:       pushMetricsData,
	}, nil
}

type metricsSenderWithObservability struct {
	exporterName string
	nextSender   requestSender
}

func (mewo *metricsSenderWithObservability) send(req request) (int, error) {
	req.setContext(obsreport.StartMetricsExportOp(req.context(), mewo.exporterName))
	numDroppedMetrics, err := mewo.nextSender.send(req)

	// TODO: this is not ideal: it should come from the next function itself.
	// 	temporarily loading it from internal format. Once full switch is done
	// 	to new metrics will remove this.
	mReq := req.(*metricsRequest)
	numReceivedMetrics, numPoints := mReq.md.MetricAndDataPointCount()

	obsreport.EndMetricsExportOp(req.context(), numPoints, numReceivedMetrics, numDroppedMetrics, err)
	return numReceivedMetrics, err
}
