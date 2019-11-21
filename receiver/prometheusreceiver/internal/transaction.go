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

package internal

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync/atomic"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/storage"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/observability"
)

const (
	portAttr   = "port"
	schemeAttr = "scheme"
)

var errMetricNameNotFound = errors.New("metricName not found from labels")
var errTransactionAborted = errors.New("transaction aborted")
var errNoJobInstance = errors.New("job or instance cannot be found from labels")

// A transaction is corresponding to an individual scrape operation or stale report.
// That said, whenever prometheus receiver scrapped a target metric endpoint a page of raw metrics is returned,
// a transaction, which acts as appender, is created to process this page of data, the scrapeLoop will call the Add or
// AddFast method to insert metrics data points, when finished either Commit, which means success, is called and data
// will be flush to the downstream consumer, or Rollback, which means discard all the data, is called and all data
// points are discarded.
type transaction struct {
	id                 int64
	ctx                context.Context
	isNew              bool
	sink               consumer.MetricsConsumer
	job                string
	instance           string
	jobsMap            *JobsMap
	useStartTimeMetric bool
	receiverName       string
	ms                 MetadataService
	node               *commonpb.Node
	metricBuilder      *metricBuilder
	logger             *zap.Logger
}

func newTransaction(ctx context.Context, jobsMap *JobsMap, useStartTimeMetric bool, receiverName string, ms MetadataService, sink consumer.MetricsConsumer, logger *zap.Logger) *transaction {
	return &transaction{
		id:                 atomic.AddInt64(&idSeq, 1),
		ctx:                ctx,
		isNew:              true,
		sink:               sink,
		jobsMap:            jobsMap,
		useStartTimeMetric: useStartTimeMetric,
		receiverName:       receiverName,
		ms:                 ms,
		logger:             logger,
	}
}

// ensure *transaction has implemented the storage.Appender interface
var _ storage.Appender = (*transaction)(nil)

// there's no document on the first return value, however, it's somehow used in AddFast. I assume this is like a
// uniqKey kind of thing for storage like a database, so that the operation can be perform faster with this key.
// however, in this case, return 0 like what the prometheus remote store does shall be enough
func (tr *transaction) Add(l labels.Labels, t int64, v float64) (uint64, error) {
	return 0, tr.AddFast(l, 0, t, v)
}

// returning an error from this method can cause the whole appending transaction to be aborted and fail
func (tr *transaction) AddFast(ls labels.Labels, _ uint64, t int64, v float64) error {
	// Important, must handle. prometheus will still try to feed the appender some data even if it failed to
	// scrape the remote target,  if the previous scrape was success and some data were cached internally
	// in our case, we don't need these data, simply drop them shall be good enough. more details:
	// https://github.com/prometheus/prometheus/blob/851131b0740be7291b98f295567a97f32fffc655/scrape/scrape.go#L933-L935
	if math.IsNaN(v) {
		return nil
	}

	select {
	case <-tr.ctx.Done():
		return errTransactionAborted
	default:
	}

	if tr.isNew {
		if err := tr.initTransaction(ls); err != nil {
			return err
		}
	}
	return tr.metricBuilder.AddDataPoint(ls, t, v)
}

func (tr *transaction) initTransaction(ls labels.Labels) error {
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
	tr.node = createNode(job, instance, mc.SharedLabels().Get(model.SchemeLabel))
	tr.metricBuilder = newMetricBuilder(mc, tr.useStartTimeMetric, tr.logger)
	tr.isNew = false
	return nil
}

// submit metrics data to consumers
func (tr *transaction) Commit() error {
	if tr.isNew {
		// In a situation like not able to connect to the remote server, scrapeloop will still commit even if it had
		// never added any data points, that the transaction has not been initialized.
		return nil
	}

	metrics, numTimeseries, droppedTimeseries, err := tr.metricBuilder.Build()
	observability.RecordMetricsForMetricsReceiver(tr.ctx, numTimeseries, droppedTimeseries)
	if err != nil {
		return err
	}

	if tr.metricBuilder.hasInternalMetric {
		m := ochttp.ClientRoundtripLatency.M(tr.metricBuilder.scrapeLatencyMs)
		stats.RecordWithTags(tr.ctx, []tag.Mutator{
			tag.Upsert(observability.TagKeyReceiver, tr.receiverName),
			tag.Upsert(ochttp.KeyClientStatus, tr.metricBuilder.scrapeStatus),
		}, m)

	}

	if tr.useStartTimeMetric {
		// AdjustStartTime - startTime has to be non-zero in this case.
		if tr.metricBuilder.startTime == 0.0 {
			metrics = []*metricspb.Metric{}
			droppedTimeseries = numTimeseries
		} else {
			adjustStartTime(tr.metricBuilder.startTime, metrics)
		}
	} else {
		// AdjustMetrics - jobsMap has to be non-nil in this case.
		// Note: metrics could be empty after adjustment, which needs to be checked before passing it on to ConsumeMetricsData()
		dropped := 0
		metrics, dropped = NewMetricsAdjuster(tr.jobsMap.get(tr.job, tr.instance), tr.logger).AdjustMetrics(metrics)
		droppedTimeseries += dropped
	}

	observability.RecordMetricsForMetricsReceiver(tr.ctx, numTimeseries, droppedTimeseries)
	if len(metrics) > 0 {
		md := consumerdata.MetricsData{
			Node:    tr.node,
			Metrics: metrics,
		}
		return tr.sink.ConsumeMetricsData(tr.ctx, md)
	}
	return nil
}

func (tr *transaction) Rollback() error {
	return nil
}

func adjustStartTime(startTime float64, metrics []*metricspb.Metric) {
	startTimeTs := timestampFromFloat64(startTime)
	for _, metric := range metrics {
		switch metric.GetMetricDescriptor().GetType() {
		case metricspb.MetricDescriptor_GAUGE_DOUBLE, metricspb.MetricDescriptor_GAUGE_DISTRIBUTION:
			continue
		default:
			for _, ts := range metric.GetTimeseries() {
				ts.StartTimestamp = startTimeTs
			}
		}
	}
}

func timestampFromFloat64(ts float64) *timestamp.Timestamp {
	secs := int64(ts)
	nanos := int64((ts - float64(secs)) * 1e9)
	return &timestamp.Timestamp{
		Seconds: secs,
		Nanos:   int32(nanos),
	}
}

func createNode(job, instance, scheme string) *commonpb.Node {
	splitted := strings.Split(instance, ":")
	host, port := splitted[0], "80"
	if len(splitted) >= 2 {
		port = splitted[1]
	}
	return &commonpb.Node{
		ServiceInfo: &commonpb.ServiceInfo{Name: job},
		Identifier: &commonpb.ProcessIdentifier{
			HostName: host,
		},
		Attributes: map[string]string{
			portAttr:   port,
			schemeAttr: scheme,
		},
	}
}
