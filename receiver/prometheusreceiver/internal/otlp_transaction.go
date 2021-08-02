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

package internal

import (
	"context"
	"errors"
	"math"
	"sync/atomic"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/obsreport"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/exemplar"
	"github.com/prometheus/prometheus/pkg/labels"
)

const (
	portAttr     = "port"
	schemeAttr   = "scheme"
	jobAttr      = "job"
	instanceAttr = "instance"

	transport  = "http"
	dataformat = "prometheus"
)

var errMetricNameNotFound = errors.New("metricName not found from labels")
var errTransactionAborted = errors.New("transaction aborted")
var errNoJobInstance = errors.New("job or instance cannot be found from labels")
var errNoStartTimeMetrics = errors.New("process_start_time_seconds metric is missing")

type transactionPdata struct {
	id                   int64
	startTimeMs          int64
	isNew                bool
	ctx                  context.Context
	useStartTimeMetric   bool
	startTimeMetricRegex string
	sink                 consumer.Metrics
	metadataService      *metadataService
	externalLabels       labels.Labels
	nodeResource         *pdata.Resource
	stalenessStore       *stalenessStore
	logger               *zap.Logger
	receiverID           config.ComponentID
	metricBuilder        *metricBuilderPdata
	job, instance        string
	jobsMap              *JobsMapPdata
	obsrecv              *obsreport.Receiver
}

type txConfig struct {
	jobsMap              *JobsMapPdata
	useStartTimeMetric   bool
	startTimeMetricRegex string
	receiverID           config.ComponentID
	ms                   *metadataService
	sink                 consumer.Metrics
	externalLabels       labels.Labels
	logger               *zap.Logger
	stalenessStore       *stalenessStore
}

func newTransactionPdata(ctx context.Context, txc *txConfig) *transactionPdata {
	return &transactionPdata{
		id:                   atomic.AddInt64(&idSeq, 1),
		ctx:                  ctx,
		isNew:                true,
		sink:                 txc.sink,
		jobsMap:              txc.jobsMap,
		useStartTimeMetric:   txc.useStartTimeMetric,
		startTimeMetricRegex: txc.startTimeMetricRegex,
		receiverID:           txc.receiverID,
		metadataService:      txc.ms,
		externalLabels:       txc.externalLabels,
		logger:               txc.logger,
		obsrecv:              obsreport.NewReceiver(obsreport.ReceiverSettings{ReceiverID: txc.receiverID, Transport: transport}),
		stalenessStore:       txc.stalenessStore,
		startTimeMs:          -1,
	}
}

// Append always returns 0 to disable label caching.
func (t *transactionPdata) Append(ref uint64, labels labels.Labels, atMs int64, value float64) (pointCount uint64, err error) {
	if t.startTimeMs < 0 {
		t.startTimeMs = atMs
	}
	if math.IsNaN(value) {
		return 0, nil
	}

	select {
	case <-t.ctx.Done():
		return 0, errTransactionAborted
	default:
	}

	if len(t.externalLabels) != 0 {
		labels = append(labels, t.externalLabels...)
	}

	if t.isNew {
		if err := t.initTransaction(labels); err != nil {
			return 0, err
		}
	}

	return 0, t.metricBuilder.AddDataPoint(labels, atMs, value)
}

func (t *transactionPdata) AppendExemplar(ref uint64, l labels.Labels, e exemplar.Exemplar) (uint64, error) {
	return 0, nil
}

func (t *transactionPdata) initTransaction(labels labels.Labels) error {
	job, instance := labels.Get(model.JobLabel), labels.Get(model.InstanceLabel)
	if job == "" || instance == "" {
		return errNoJobInstance
	}
	metadataCache, err := t.metadataService.Get(job, instance)
	if err != nil {
		return err
	}
	if t.jobsMap != nil {
		t.job = job
		t.instance = instance
	}
	t.nodeResource = createNodeAndResourcePdata(job, instance, metadataCache.SharedLabels().Get(model.SchemeLabel))
	t.metricBuilder = newMetricBuilderPdata(metadataCache, t.useStartTimeMetric, t.startTimeMetricRegex, t.logger, t.stalenessStore)
	t.isNew = false
	return nil
}

func (t *transactionPdata) Commit() error {
	if t.isNew {
		return nil
	}

	// Emit the staleness markers.
	staleLabels := t.stalenessStore.emitStaleLabels()
	for _, sEntry := range staleLabels {
		t.metricBuilder.AddDataPoint(sEntry.labels, sEntry.seenAtMs, stalenessSpecialValue)
	}
	t.startTimeMs = -1

	ctx := t.obsrecv.StartMetricsOp(t.ctx)
	metricsL, numPoints, _, err := t.metricBuilder.Build()
	if err != nil {
		t.obsrecv.EndMetricsOp(ctx, dataformat, 0, err)
		return err
	}

	if t.useStartTimeMetric {
		if t.metricBuilder.startTime == 0.0 {
			err = errNoStartTimeMetrics
			t.obsrecv.EndMetricsOp(ctx, dataformat, 0, err)
			return err
		}
		// Otherwise adjust the startTimestamp for all the metrics.
		adjustStartTimestampPdata(t.metricBuilder.startTime, metricsL)
	} else {
		// TODO: Derive numPoints in this case.
		_ = NewMetricsAdjusterPdata(t.jobsMap.get(t.job, t.instance), t.logger).AdjustMetrics(metricsL)
	}

	if metricsL.Len() > 0 {
		metrics := t.metricSliceToMetrics(metricsL)
		t.sink.ConsumeMetrics(ctx, *metrics)
	}

	t.obsrecv.EndMetricsOp(ctx, dataformat, numPoints, nil)
	return nil
}

func (t *transactionPdata) Rollback() error {
	t.startTimeMs = -1
	return nil
}

func adjustStartTimestampPdata(startTime float64, metricsL *pdata.MetricSlice) {
	for i := 0; i < metricsL.Len(); i++ {
		metric := metricsL.At(i)
		switch metric.DataType() {
		case pdata.MetricDataTypeGauge, pdata.MetricDataTypeHistogram:
			continue

		default:
			dataPoints := metric.Summary().DataPoints()
			for i := 0; i < dataPoints.Len(); i++ {
				dataPoint := dataPoints.At(i)
				dataPoint.SetStartTimestamp(pdata.Timestamp(startTime))
			}
		}
	}
}

func (t *transactionPdata) metricSliceToMetrics(metricsL *pdata.MetricSlice) *pdata.Metrics {
	metrics := pdata.NewMetrics()
	rms := metrics.ResourceMetrics().AppendEmpty()
	ilm := rms.InstrumentationLibraryMetrics().AppendEmpty()
	metricsL.CopyTo(ilm.Metrics())
	t.nodeResource.CopyTo(rms.Resource())
	return &metrics
}
