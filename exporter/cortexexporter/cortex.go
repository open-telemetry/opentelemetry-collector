// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cortexexporter

import (
	"context"
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	"net/http"
	"strconv"
	"strings"
	"sync"
)
// TODO: get default labels such as job or instance from Resource

// cortexExporter converts OTLP metrics to Cortex TimeSeries and sends them to a remote endpoint
type cortexExporter struct {
	namespace 	string
	endpoint  	string
	client    	*http.Client
	headers		map[string]string
	wg		  	*sync.WaitGroup
	closeChan	chan struct{}
}

// handleScalarMetric processes data points in a single OTLP scalar metric by adding the each point as a Sample into
// its corresponding TimeSeries in tsMap.
// tsMap and metric cannot be nil, and metric must have a non-nil descriptor
func (ce *cortexExporter) handleScalarMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	mType := metric.MetricDescriptor.Type
	switch mType {
	// int points
	case otlp.MetricDescriptor_MONOTONIC_INT64,otlp.MetricDescriptor_INT64:
		if metric.Int64DataPoints == nil {
			return fmt.Errorf("nil data point field in metric" + metric.GetMetricDescriptor().Name)
		}
		for _, pt := range metric.Int64DataPoints {
			name := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)
			lbs := createLabelSet(pt.GetLabels(), nameStr, name)
			sample := &prompb.Sample{
				Value: 		float64(pt.Value),
				Timestamp: 	int64(pt.TimeUnixNano),
			}
			addSample(tsMap,sample, lbs, metric.GetMetricDescriptor().GetType())
		}
		return nil

	// double points
	case otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
		if metric.DoubleDataPoints == nil {
			return fmt.Errorf("nil data point field in metric" + metric.GetMetricDescriptor().Name)
		}
		for _, pt := range metric.DoubleDataPoints {
			name := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)
			lbs := createLabelSet(pt.GetLabels(),nameStr, name)
			sample := &prompb.Sample{
				Value: 		pt.Value,
				Timestamp: 	int64(pt.TimeUnixNano),
			}
			addSample(tsMap,sample, lbs, metric.GetMetricDescriptor().GetType())
		}
		return nil
	}

	return fmt.Errorf("invalid metric type: wants int or double data points");
}

// handleHistogramMetric processes data points in a single OTLP histogram metric by mapping the sum, count and each
// bucket of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (ce *cortexExporter) handleHistogramMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	if metric.HistogramDataPoints == nil {
		return fmt.Errorf("invalid metric type: wants histogram points")
	}
	for _, pt := range metric.HistogramDataPoints {
		time := int64(pt.GetTimeUnixNano())
		ty := metric.GetMetricDescriptor().GetType()
		baseName := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)
		sum := &prompb.Sample{
			Value:		pt.GetSum(),
			Timestamp:	time,
		}
		count := &prompb.Sample{
			Value:		float64(pt.GetCount()),
			Timestamp:	time,
		}
		sumLbs := createLabelSet(pt.GetLabels(),nameStr, baseName+sumStr)
		countLbs := createLabelSet(pt.GetLabels(),nameStr, baseName+countStr)
		addSample(tsMap, sum, sumLbs, ty)
		addSample(tsMap, count, countLbs, ty)
		var totalCount uint64
		for le, bk := range pt.GetBuckets(){
			bucket := &prompb.Sample{
				Value:		float64(bk.Count),
				Timestamp:	time,
			}
			boundStr := strconv.FormatFloat(pt.GetExplicitBounds()[le], 'f',-1, 64)
			lbs := createLabelSet(pt.GetLabels(),nameStr, baseName+bucketStr, leStr,boundStr)
			addSample(tsMap, bucket, lbs ,ty)
			totalCount += bk.GetCount()
		}
		infSample := &prompb.Sample{Value:float64(totalCount),Timestamp:time}
		infLbs := createLabelSet(pt.GetLabels(),nameStr, baseName+bucketStr, leStr,pInfStr)
		addSample(tsMap, infSample, infLbs, ty)
	}
	return nil
}

// handleSummaryMetric processes data points in a single OTLP summary metric by mapping the sum, count and each
// quantile of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (ce *cortexExporter) handleSummaryMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	if metric.SummaryDataPoints == nil {
		return fmt.Errorf("invalid metric type: wants summary points")
	}

	for _, pt := range metric.SummaryDataPoints {
		time := int64(pt.GetTimeUnixNano())
		ty := metric.GetMetricDescriptor().GetType()
		baseName := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)
		sum := &prompb.Sample{
			Value:		pt.GetSum(),
			Timestamp:	time,
		}
		count := &prompb.Sample{
			Value:		float64(pt.GetCount()),
			Timestamp:	time,
		}
		sumLbs := createLabelSet(pt.GetLabels(),nameStr, baseName+sumStr)
		countLbs := createLabelSet(pt.GetLabels(),nameStr, baseName+countStr)
		addSample(tsMap, sum, sumLbs, ty)
		addSample(tsMap, count, countLbs, ty)
		for _, qt := range pt.GetPercentileValues(){
			quantile := &prompb.Sample{
				Value:		float64(qt.Value),
				Timestamp:	time,
			}
			percentileStr := strconv.FormatFloat(qt.Percentile, 'f',-1, 64)
			qtLbs := createLabelSet(pt.GetLabels(),nameStr, baseName, quantileStr, percentileStr)
			addSample(tsMap, quantile, qtLbs, ty)
		}
	}
	return nil

}

// newCortexExporter initializes a new cortexExporter instance and sets fields accordingly.
// client parameter cannot be nil.
func newCortexExporter(ns string, ep string, client *http.Client) (*cortexExporter, error) {
	if client == nil {
		return nil, fmt.Errorf("http client cannot be nil")
	}

	return &cortexExporter{
		namespace: 	ns,
		endpoint:  	ep,
		client:    	client,
		wg:			new(sync.WaitGroup),
		closeChan:  make(chan struct{}),
	}, nil
}

// shutdown stops the exporter from accepting incoming calls(and return error), and wait for current export operations
// to finish before returning
func (ce *cortexExporter)shutdown(context.Context) error{
	close(ce.closeChan)
	ce.wg.Wait()
	return nil
}

// pushMetrics converts metrics to Cortex TimeSeries and send to remote endpoint. It maintain a map of TimeSeries,
// validates and handles each individual metric, adding the converted TimeSeries to the map, and finally
// exports the map.
func (ce *cortexExporter) pushMetrics(ctx context.Context, md pdata.Metrics) (int, error) {
	ce.wg.Add(1)
	defer ce.wg.Done()
	select{
	case <-ce.closeChan:
		return pdatautil.MetricCount(md),fmt.Errorf("shutdown has been called")
	default:
		tsMap := map[string]*prompb.TimeSeries{}
		dropped := 0
		errStrings := []string{}
		rms := data.MetricDataToOtlp(pdatautil.MetricsToInternalMetrics(md))
		for _, r := range rms {
			// TODO add resource attributes as labels
			for _, inst := range r.InstrumentationLibraryMetrics {
				//TODO add instrumentation library information as labels
				for _, m := range inst.Metrics {
					ok := validateMetrics(m.MetricDescriptor)
					if !ok {
						dropped++
						errStrings = append(errStrings, "invalid temporality and type combination")
						continue
					}
					switch m.GetMetricDescriptor().GetType() {
					case otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_INT64,
						otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
							ce.handleScalarMetric(tsMap,m)
					case otlp.MetricDescriptor_HISTOGRAM:
							ce.handleHistogramMetric(tsMap,m)
					case otlp.MetricDescriptor_SUMMARY:
							ce.handleSummaryMetric(tsMap,m)
					default:
						dropped++
						errStrings = append(errStrings, "invalid type")
						continue
					}
				}
			}
		}
		if(dropped != 0) {
			return dropped, fmt.Errorf(strings.Join(errStrings, "\n"))
		}

		if err := ce.export(ctx,tsMap); err != nil {
			return pdatautil.MetricCount(md), err
		}

		return 0, nil
	}
}

// export sends TimeSeries in tsMap to a Cortex Gateway
// this needs to be done
func (ce *cortexExporter) export(ctx context.Context, tsMap map[string]*prompb.TimeSeries) error {
	return nil
}
