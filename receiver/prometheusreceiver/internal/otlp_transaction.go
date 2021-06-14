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
	"math"
	"sync/atomic"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// A transaction is corresponding to an individual scrape operation or stale report.
// That said, whenever prometheus receiver scrapped a target metric endpoint a page of raw metrics is returned,
// a transaction, which acts as appender, is created to process this page of data, the scrapeLoop will call the Add or
// AddFast method to insert metrics data points, when finished either Commit, which means success, is called and data
// will be flush to the downstream consumer, or Rollback, which means discard all the data, is called and all data
// points are discarded.
type transactionPdata struct {
	*transaction
	resource      pdata.Resource
	metricBuilder *metricBuilderPdata
}

func newTransactionPdata(
	ctx context.Context,
	jobsMap *JobsMap,
	useStartTimeMetric bool,
	startTimeMetricRegex string,
	receiverID config.ComponentID,
	ms *metadataService,
	sink consumer.Metrics,
	externalLabels labels.Labels,
	logger *zap.Logger) *transactionPdata {
	return &transactionPdata{
		transaction: &transaction{
			id:                   atomic.AddInt64(&idSeq, 1),
			ctx:                  ctx,
			isNew:                true,
			sink:                 sink,
			jobsMap:              jobsMap,
			useStartTimeMetric:   useStartTimeMetric,
			startTimeMetricRegex: startTimeMetricRegex,
			receiverID:           receiverID,
			ms:                   ms,
			externalLabels:       externalLabels,
			logger:               logger,
			obsrecv:              obsreport.NewReceiver(obsreport.ReceiverSettings{ReceiverID: receiverID, Transport: transport}),
		},
	}
}

// ensure *transaction has implemented the storage.Appender interface
var _ storage.Appender = (*transactionPdata)(nil)

// Append always returns 0 to disable label caching.
func (tr *transactionPdata) Append(ref uint64, ls labels.Labels, t int64, v float64) (uint64, error) {
	// Important, must handle. prometheus will still try to feed the appender some data even if it failed to
	// scrape the remote target,  if the previous scrape was success and some data were cached internally
	// in our case, we don't need these data, simply drop them shall be good enough. more details:
	// https://github.com/prometheus/prometheus/blob/851131b0740be7291b98f295567a97f32fffc655/scrape/scrape.go#L933-L935
	if math.IsNaN(v) {
		return 0, nil
	}

	select {
	case <-tr.ctx.Done():
		return 0, errTransactionAborted
	default:
	}
	if len(tr.externalLabels) > 0 {
		// TODO(jbd): Improve the allocs.
		ls = append(ls, tr.externalLabels...)
	}
	if tr.isNew {
		if err := tr.initTransaction(ls); err != nil {
			return 0, err
		}
	}
	return 0, tr.metricBuilder.AddDataPoint(ls, t, v)
}

func (tr *transactionPdata) initTransaction(ls labels.Labels) error {
	job, instance := ls.Get(model.JobLabel), ls.Get(model.InstanceLabel)
	if job == "" || instance == "" {
		return errNoJobInstance
	}
	// discover the binding target when this method is called for the first time during a transaction
	mc, err := tr.ms.Get(job, instance)
	if err != nil {
		return err
	}
	if tr.jobsMap != nil {
		tr.job = job
		tr.instance = instance
	}
	tr.resource = createNodeAndResourcePdata(job, instance, mc.SharedLabels().Get(model.SchemeLabel))
	tr.metricBuilder = newMetricBuilderPdata(mc, tr.useStartTimeMetric, tr.startTimeMetricRegex, tr.logger)
	tr.isNew = false
	return nil
}

// Commit submits metrics data to consumers.
func (tr *transactionPdata) Commit() error {
	if tr.isNew {
		// In a situation like not able to connect to the remote server, scrapeloop will still commit even if it had
		// never added any data points, that the transaction has not been initialized.
		return nil
	}

	ctx := tr.obsrecv.StartMetricsOp(tr.ctx)
	metrics, _, _, err := tr.metricBuilder.Build()
	if err != nil {
		// Only error by Build() is errNoDataToBuild, with numReceivedPoints set to zero.
		tr.obsrecv.EndMetricsOp(ctx, dataformat, 0, err)
		return err
	}

	if tr.useStartTimeMetric {
		// startTime is mandatory in this case, but may be zero when the
		// process_start_time_seconds metric is missing from the target endpoint.
		if tr.metricBuilder.startTime == 0.0 {
			// Since we are unable to adjust metrics properly, we will drop them
			// and return an error.
			err = errNoStartTimeMetrics
			tr.obsrecv.EndMetricsOp(ctx, dataformat, 0, err)
			return err
		}

		adjustStartTimestampPdata(tr.metricBuilder.startTime, metrics)
	} else {
		// AdjustMetrics - jobsMap has to be non-nil in this case.
		// Note: metrics could be empty after adjustment, which needs to be checked before passing it on to ConsumeMetrics()
		metrics, _ = NewMetricsAdjuster(tr.jobsMap.get(tr.job, tr.instance), tr.logger).AdjustMetrics(metrics)
	}

	numPoints := 0
	if len(metrics) > 0 {
                numPoints = 
		err = tr.sink.ConsumeMetrics(ctx, metrics)
	}
	tr.obsrecv.EndMetricsOp(ctx, dataformat, numPoints, err)
	return err
}

func adjustStartTimestampPdata(startTime float64, metrics []*pdata.Metric) {
	startTimeTs := pdata.Timestamp(startTime)
	for _, metric := range metrics {
		switch metric.MetricDataType() {
		case pdata.MetricDataTypeDoubleGauge, pdata.MetricDataTypeHistogram:
			continue

		// For all the other metrics, update the startTimestamp to startTimeTs.

		case pdata.MetricDataTypeIntHistogram:
			dataPoints := metric.IntHistogram().DataPoints()
			for i := 0; i < dataPoints.Len(); i++ {
				point := dataPoints.At(i)
				point.SetStartTimestamp(startTimeTs)
			}

		case pdata.MetricDataTypeIntGauge:
			dataPoints := metric.IntGauge().DataPoints()
			for i := 0; i < dataPoints.Len(); i++ {
				point := dataPoints.At(i)
				point.SetStartTimestamp(startTimeTs)
			}

		case pdata.MetricDataTypeIntSum:
			dataPoints := metric.IntSum().DataPoints()
			for i := 0; i < dataPoints.Len(); i++ {
				point := dataPoints.At(i)
				point.SetStartTimestamp(startTimeTs)
			}

		case pdata.MetricDataTypeDoubleSum:
			dataPoints := metric.DoubleSum().DataPoints()
			for i := 0; i < dataPoints.Len(); i++ {
				point := dataPoints.At(i)
				point.SetStartTimestamp(startTimeTs)
			}

		case pdata.MetricDataTypeIntHistogram:
			dataPoints := metric.IntHistogram().DataPoints()
			for i := 0; i < dataPoints.Len(); i++ {
				point := dataPoints.At(i)
				point.SetStartTimestamp(startTimeTs)
			}

		case pdata.MetricDataTypeSummary:
			dataPoints := metric.Summary().DataPoints()
			for i := 0; i < dataPoints.Len(); i++ {
				point := dataPoints.At(i)
				point.SetStartTimestamp(startTimeTs)
			}
		}
	}
}
