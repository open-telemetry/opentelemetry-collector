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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

// TODO: get default labels such as job or instance from Resource

// cortexExporter converts OTLP metrics to Cortex TimeSeries and sends them to a remote endpoint
type cortexExporter struct {
	namespace string
	endpoint  string
	client    *http.Client
	headers   map[string]string
	wg        *sync.WaitGroup
	closeChan chan struct{}
}

// handleScalarMetric processes data points in a single OTLP scalar metric by adding the each point as a Sample into
// its corresponding TimeSeries in tsMap.
// tsMap and metric cannot be nil, and metric must have a non-nil descriptor
func (ce *cortexExporter) handleScalarMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {

	mType := metric.MetricDescriptor.Type

	switch mType {
	// int points
	case otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_INT64:
		if metric.Int64DataPoints == nil {
			return fmt.Errorf("nil data point field in metric" + metric.GetMetricDescriptor().Name)
		}

		for _, pt := range metric.Int64DataPoints {

			// create parameters for addSample
			name := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)
			lbs := createLabelSet(pt.GetLabels(), nameStr, name)
			sample := &prompb.Sample{
				Value:     float64(pt.Value),
				Timestamp: int64(pt.TimeUnixNano),
			}

			addSample(tsMap, sample, lbs, metric.GetMetricDescriptor().GetType())
		}
		return nil

	// double points
	case otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
		if metric.DoubleDataPoints == nil {
			return fmt.Errorf("nil data point field in metric" + metric.GetMetricDescriptor().Name)
		}
		for _, pt := range metric.DoubleDataPoints {

			// create parameters for addSample
			name := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)
			lbs := createLabelSet(pt.GetLabels(), nameStr, name)
			sample := &prompb.Sample{
				Value:     pt.Value,
				Timestamp: int64(pt.TimeUnixNano),
			}

			addSample(tsMap, sample, lbs, metric.GetMetricDescriptor().GetType())
		}
		return nil
	}

	return fmt.Errorf("invalid metric type: wants int or double data points")
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

		// sum, count, and buckets of the histogram should append suffix to baseName
		baseName := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)

		// treat sum as sample in an individual TimeSeries
		sum := &prompb.Sample{
			Value:     pt.GetSum(),
			Timestamp: time,
		}
		sumLbs := createLabelSet(pt.GetLabels(), nameStr, baseName+sumStr)
		addSample(tsMap, sum, sumLbs, ty)

		// treat count as a sample in an individual TimeSeries
		count := &prompb.Sample{
			Value:     float64(pt.GetCount()),
			Timestamp: time,
		}
		countLbs := createLabelSet(pt.GetLabels(), nameStr, baseName+countStr)
		addSample(tsMap, count, countLbs, ty)

		// count for +Inf bound
		var totalCount uint64

		for le, bk := range pt.GetBuckets() {
			bucket := &prompb.Sample{
				Value:     float64(bk.Count),
				Timestamp: time,
			}
			boundStr := strconv.FormatFloat(pt.GetExplicitBounds()[le], 'f', -1, 64)
			lbs := createLabelSet(pt.GetLabels(), nameStr, baseName+bucketStr, leStr, boundStr)
			addSample(tsMap, bucket, lbs, ty)

			totalCount += bk.GetCount()
		}

		infBucket := &prompb.Sample{
			Value: float64(totalCount),
			Timestamp: time,
		}
		infLbs := createLabelSet(pt.GetLabels(), nameStr, baseName+bucketStr, leStr, pInfStr)
		addSample(tsMap, infBucket, infLbs, ty)
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

		// sum and count of the Summary should append suffix to baseName
		baseName := getPromMetricName(metric.GetMetricDescriptor(), ce.namespace)

		// treat sum as sample in an individual TimeSeries
		sum := &prompb.Sample{
			Value:     pt.GetSum(),
			Timestamp: time,
		}
		sumLbs := createLabelSet(pt.GetLabels(), nameStr, baseName+sumStr)
		addSample(tsMap, sum, sumLbs, ty)

		// treat count as a sample in an individual TimeSeries
		count := &prompb.Sample{
			Value:     float64(pt.GetCount()),
			Timestamp: time,
		}
		countLbs := createLabelSet(pt.GetLabels(), nameStr, baseName+countStr)
		addSample(tsMap, count, countLbs, ty)

		for _, qt := range pt.GetPercentileValues() {
			quantile := &prompb.Sample{
				Value:     float64(qt.Value),
				Timestamp: time,
			}
			percentileStr := strconv.FormatFloat(qt.Percentile, 'f', -1, 64)
			qtLbs := createLabelSet(pt.GetLabels(), nameStr, baseName, quantileStr, percentileStr)
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
		namespace: ns,
		endpoint:  ep,
		client:    client,
		wg:        new(sync.WaitGroup),
		closeChan: make(chan struct{}),
	}, nil
}

// shutdown stops the exporter from accepting incoming calls(and return error), and wait for current export operations
// to finish before returning
func (ce *cortexExporter) shutdown(context.Context) error {
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
	select {
	case <-ce.closeChan:
		return pdatautil.MetricCount(md), fmt.Errorf("shutdown has been called")
	default:
		// stores TimeSeries
		tsMap := map[string]*prompb.TimeSeries{}
		dropped := 0
		errs := []string{}

		rms := data.MetricDataToOtlp(pdatautil.MetricsToInternalMetrics(md))

		for _, r := range rms {
			// TODO: add resource attributes as labels
			for _, inst := range r.InstrumentationLibraryMetrics {
				//TODO: add instrumentation library information as labels
				for _, m := range inst.Metrics {
					// check for valid type and temporality combination
					ok := validateMetrics(m.MetricDescriptor)
					if !ok {
						dropped++
						errs = append(errs, "invalid temporality and type combination")
						continue
					}
					// handle individual metric based on type
					switch m.GetMetricDescriptor().GetType() {
						case otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_INT64,
							otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
							if err := ce.handleScalarMetric(tsMap, m); err != nil {
								errs = append(errs, err.Error())
							}
						case otlp.MetricDescriptor_HISTOGRAM:
							if err := ce.handleHistogramMetric(tsMap, m); err != nil {
								errs = append(errs, err.Error())
							}
						case otlp.MetricDescriptor_SUMMARY:
							if err := ce.handleSummaryMetric(tsMap, m); err != nil {
								errs = append(errs, err.Error())
							}
						default:
							dropped++
							errs = append(errs, "invalid type")
							continue
					}
				}
			}
		}

		if err := ce.Export(ctx, tsMap); err != nil {
			return pdatautil.MetricCount(md), err
		}

		if dropped != 0 {
			return dropped, fmt.Errorf(strings.Join(errs, "\n"))
		}

		return 0, nil
	}
}

/*
Because we are adhering closely to the Remote Write API, we must Export a
Snappy-compressed WriteRequest instance of the TimeSeries Metrics in order
for the Remote Write Endpoint to properly receive our Metrics data.
*/
func (ce *cortexExporter) Export(ctx context.Context, TsMap map[string]*prompb.TimeSeries) error {
	//Calls the helper function to convert the TsMap to the desired format
	req, err := wrapTimeSeries(TsMap)
	if err != nil {
		return err
	}

	//Uses proto.Marshal to convert the WriteRequest into wire format (bytes array)
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	//Makes use of the snappy compressor, as we are emulating the Remote Write package
	compressedData := snappy.Encode(nil, data)

	//Create the HTTP POST request to send to the endpoint
	httpReq, err := http.NewRequest("POST", ce.endpoint, bytes.NewReader(compressedData))
	if err != nil {
		return err
	}

	//Add optional headers
	for name, value := range ce.headers {
		httpReq.Header.Set(name, value)
	}

	//Add necessary headers
	httpReq.Header.Add("Content-Encoding", "snappy")
	httpReq.Header.Add("Accept-Encoding", "snappy")
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	//Changing context of the httpreq to global context
	httpReq = httpReq.WithContext(ctx)

	ctx, cancel := context.WithTimeout(context.Background(), ce.client.Timeout)
	defer cancel()

	httpResp, err := ce.client.Do(httpReq)
	if err != nil {
		return err
	}

	//Only even status codes < 400 are okay
	if httpResp.StatusCode/100 != 2 || httpResp.StatusCode >= 400 {
		scanner := bufio.NewScanner(io.LimitReader(httpResp.Body, 256))
		line := ""
		if scanner.Scan() {
			line = scanner.Text()
		}
		err = errors.Errorf("server returned HTTP status %s: %s", httpResp.Status, line)
	}
	return err
}
