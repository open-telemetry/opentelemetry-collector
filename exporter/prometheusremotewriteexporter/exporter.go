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
package prometheusremotewriteexporter

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

// prwExporter converts OTLP metrics to Prometheus remote write TimeSeries and sends them to a remote endpoint
type prwExporter struct {
	namespace   string
	endpointURL *url.URL
	client      *http.Client
	wg          *sync.WaitGroup
	closeChan   chan struct{}
}

// newPrwExporter initializes a new prwExporter instance and sets fields accordingly.
// client parameter cannot be nil.
func newPrwExporter(namespace string, endpoint string, client *http.Client) (*prwExporter, error) {

	if client == nil {
		return nil, errors.New("http client cannot be nil")
	}

	endpointURL, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, errors.New("invalid endpoint")
	}

	return &prwExporter{
		namespace:   namespace,
		endpointURL: endpointURL,
		client:      client,
		wg:          new(sync.WaitGroup),
		closeChan:   make(chan struct{}),
	}, nil
}

// shutdown stops the exporter from accepting incoming calls(and return error), and wait for current export operations
// to finish before returning
func (prwe *prwExporter) shutdown(context.Context) error {
	close(prwe.closeChan)
	prwe.wg.Wait()
	return nil
}

// pushMetrics converts metrics to Prometheus remote write TimeSeries and send to remote endpoint. It maintain a map of
// TimeSeries, validates and handles each individual metric, adding the converted TimeSeries to the map, and finally
// exports the map.
func (prwe *prwExporter) pushMetrics(ctx context.Context, md pdata.Metrics) (int, error) {

	prwe.wg.Add(1)
	defer prwe.wg.Done()
	select {
	case <-prwe.closeChan:
		return pdatautil.MetricCount(md), errors.New("shutdown has been called")
	default:
		tsMap := map[string]*prompb.TimeSeries{}
		dropped := 0
		errs := []string{}

		resourceMetrics := data.MetricDataToOtlp(pdatautil.MetricsToInternalMetrics(md))
		for _, r := range resourceMetrics {
			for _, instMetrics := range r.InstrumentationLibraryMetrics {
				// TODO: decide if instrumentation library information should be exported as labels
				for _, metric := range instMetrics.Metrics {
					// check for valid type and temporality combination
					if ok := validateMetrics(metric.MetricDescriptor); !ok {
						dropped++
						errs = append(errs, "invalid temporality and type combination")
						continue
					}
					// handle individual metric based on type
					switch metric.GetMetricDescriptor().GetType() {
					case otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_INT64,
						otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
						if err := prwe.handleScalarMetric(tsMap, metric); err != nil {
							errs = append(errs, err.Error())
						}
					case otlp.MetricDescriptor_HISTOGRAM:
						if err := prwe.handleHistogramMetric(tsMap, metric); err != nil {
							errs = append(errs, err.Error())
						}
					case otlp.MetricDescriptor_SUMMARY:
						if err := prwe.handleSummaryMetric(tsMap, metric); err != nil {
							errs = append(errs, err.Error())
						}
					}
				}
			}
		}

		if err := prwe.export(ctx, tsMap); err != nil {
			return pdatautil.MetricCount(md), err
		}

		if dropped != 0 {
			return dropped, errors.New(strings.Join(errs, "\n"))
		}

		return 0, nil
	}
}

// handleScalarMetric processes data points in a single OTLP scalar metric by adding the each point as a Sample into
// its corresponding TimeSeries in tsMap.
// tsMap and metric cannot be nil, and metric must have a non-nil descriptor
func (prwe *prwExporter) handleScalarMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	// add check for nil

	mType := metric.MetricDescriptor.Type

	switch mType {
	// int points
	case otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_INT64:
		if metric.Int64DataPoints == nil {
			return errors.New("nil data point field in metric" + metric.GetMetricDescriptor().Name)
		}

		for _, pt := range metric.Int64DataPoints {

			// create parameters for addSample
			name := getPromMetricName(metric.GetMetricDescriptor(), prwe.namespace)
			labels := createLabelSet(pt.GetLabels(), nameStr, name)
			sample := &prompb.Sample{
				Value: float64(pt.Value),
				// convert ns to ms
				Timestamp: convertTimeStamp(pt.TimeUnixNano),
			}

			addSample(tsMap, sample, labels, metric.GetMetricDescriptor().GetType())
		}
		return nil

	// double points
	case otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
		if metric.DoubleDataPoints == nil {
			return errors.New("nil data point field in metric" + metric.GetMetricDescriptor().Name)
		}
		for _, pt := range metric.DoubleDataPoints {

			// create parameters for addSample
			name := getPromMetricName(metric.GetMetricDescriptor(), prwe.namespace)
			labels := createLabelSet(pt.GetLabels(), nameStr, name)
			sample := &prompb.Sample{
				Value:     pt.Value,
				Timestamp: convertTimeStamp(pt.TimeUnixNano),
			}

			addSample(tsMap, sample, labels, metric.GetMetricDescriptor().GetType())
		}
		return nil
	}

	return errors.New("invalid metric type: wants int or double data points")
}

// handleHistogramMetric processes data points in a single OTLP histogram metric by mapping the sum, count and each
// bucket of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *prwExporter) handleHistogramMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {

	if metric.HistogramDataPoints == nil {
		return errors.New("invalid metric type: wants histogram points")
	}

	for _, pt := range metric.HistogramDataPoints {

		time := convertTimeStamp(pt.TimeUnixNano)
		mType := metric.GetMetricDescriptor().GetType()

		// sum, count, and buckets of the histogram should append suffix to baseName
		baseName := getPromMetricName(metric.GetMetricDescriptor(), prwe.namespace)

		// treat sum as sample in an individual TimeSeries
		sum := &prompb.Sample{
			Value:     pt.GetSum(),
			Timestamp: time,
		}
		sumlabels := createLabelSet(pt.GetLabels(), nameStr, baseName+sumStr)
		addSample(tsMap, sum, sumlabels, mType)

		// treat count as a sample in an individual TimeSeries
		count := &prompb.Sample{
			Value:     float64(pt.GetCount()),
			Timestamp: time,
		}
		countlabels := createLabelSet(pt.GetLabels(), nameStr, baseName+countStr)
		addSample(tsMap, count, countlabels, mType)

		// count for +Inf bound
		var totalCount uint64

		// process each bucket
		for le, bk := range pt.GetBuckets() {
			bucket := &prompb.Sample{
				Value:     float64(bk.Count),
				Timestamp: time,
			}
			boundStr := strconv.FormatFloat(pt.GetExplicitBounds()[le], 'f', -1, 64)
			labels := createLabelSet(pt.GetLabels(), nameStr, baseName+bucketStr, leStr, boundStr)
			addSample(tsMap, bucket, labels, mType)

			totalCount += bk.GetCount()
		}

		infBucket := &prompb.Sample{
			Value:     float64(totalCount),
			Timestamp: time,
		}
		inflabels := createLabelSet(pt.GetLabels(), nameStr, baseName+bucketStr, leStr, pInfStr)
		addSample(tsMap, infBucket, inflabels, mType)
	}
	return nil
}

// handleSummaryMetric processes data points in a single OTLP summary metric by mapping the sum, count and each
// quantile of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *prwExporter) handleSummaryMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {

	if metric.SummaryDataPoints == nil {
		return errors.New("invalid metric type: wants summary points")
	}

	for _, pt := range metric.SummaryDataPoints {

		time := convertTimeStamp(pt.TimeUnixNano)
		mType := metric.GetMetricDescriptor().GetType()

		// sum and count of the Summary should append suffix to baseName
		baseName := getPromMetricName(metric.GetMetricDescriptor(), prwe.namespace)

		// treat sum as sample in an individual TimeSeries
		sum := &prompb.Sample{
			Value:     pt.GetSum(),
			Timestamp: time,
		}
		sumlabels := createLabelSet(pt.GetLabels(), nameStr, baseName+sumStr)
		addSample(tsMap, sum, sumlabels, mType)

		// treat count as a sample in an individual TimeSeries
		count := &prompb.Sample{
			Value:     float64(pt.GetCount()),
			Timestamp: time,
		}
		countlabels := createLabelSet(pt.GetLabels(), nameStr, baseName+countStr)
		addSample(tsMap, count, countlabels, mType)

		// process each percentile/quantile
		for _, qt := range pt.GetPercentileValues() {
			quantile := &prompb.Sample{
				Value:     qt.Value,
				Timestamp: time,
			}
			percentileStr := strconv.FormatFloat(qt.Percentile, 'f', -1, 64)
			qtlabels := createLabelSet(pt.GetLabels(), nameStr, baseName, quantileStr, percentileStr)
			addSample(tsMap, quantile, qtlabels, mType)
		}
	}
	return nil
}

// Because we are adhering closely to the Remote Write API, we must Export a
// Snappy-compressed WriteRequest instance of the TimeSeries Metrics in order
// for the Remote Write Endpoint to properly receive our Metrics data.
func (prwe *prwExporter) export(ctx context.Context, tsMap map[string]*prompb.TimeSeries) error {
	//Calls the helper function to convert the TsMap to the desired format
	req, err := wrapTimeSeries(tsMap)
	if err != nil {
		return err
	}

	//Uses proto.Marshal to convert the WriteRequest into wire format (bytes array)
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	buf := make([]byte, len(data), cap(data))
	//Makes use of the snappy compressor, as we are emulating the Remote Write package
	compressedData := snappy.Encode(buf, data)

	//Create the HTTP POST request to send to the endpoint
	httpReq, err := http.NewRequest("POST", prwe.endpointURL.String(), bytes.NewReader(compressedData))
	if err != nil {
		return err
	}

	// Add necessary headers specified by:
	// https://cortexmetrics.io/docs/apis/#remote-api
	httpReq.Header.Add("Content-Encoding", "snappy")
	httpReq.Header.Set("User-Agent", "otel-collector")
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	//Changing context of the httpreq to global context
	httpReq = httpReq.WithContext(ctx)

	_, cancel := context.WithTimeout(context.Background(), prwe.client.Timeout)
	defer cancel()

	httpResp, err := prwe.client.Do(httpReq)
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
		err = errors.New("server returned HTTP status: " + httpResp.Status + ", " + line)
	}
	return err
}
