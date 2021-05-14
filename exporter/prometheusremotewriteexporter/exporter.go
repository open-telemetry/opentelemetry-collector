// Copyright The OpenTelemetry Authors
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

// Package prometheusremotewriteexporter implements an exporter that sends Prometheus remote write requests.
package prometheusremotewriteexporter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	maxConcurrentRequests = 5
	maxBatchByteSize      = 3000000
)

// PrwExporter converts OTLP metrics to Prometheus remote write TimeSeries and sends them to a remote endpoint.
type PrwExporter struct {
	namespace       string
	externalLabels  map[string]string
	endpointURL     *url.URL
	client          *http.Client
	wg              *sync.WaitGroup
	closeChan       chan struct{}
	userAgentHeader string
}

// NewPrwExporter initializes a new PrwExporter instance and sets fields accordingly.
// client parameter cannot be nil.
func NewPrwExporter(namespace string, endpoint string, client *http.Client, externalLabels map[string]string, buildInfo component.BuildInfo) (*PrwExporter, error) {
	if client == nil {
		return nil, errors.New("http client cannot be nil")
	}

	sanitizedLabels, err := validateAndSanitizeExternalLabels(externalLabels)
	if err != nil {
		return nil, err
	}

	endpointURL, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, errors.New("invalid endpoint")
	}

	userAgentHeader := fmt.Sprintf("%s/%s", strings.ReplaceAll(strings.ToLower(buildInfo.Description), " ", "-"), buildInfo.Version)

	return &PrwExporter{
		namespace:       namespace,
		externalLabels:  sanitizedLabels,
		endpointURL:     endpointURL,
		client:          client,
		wg:              new(sync.WaitGroup),
		closeChan:       make(chan struct{}),
		userAgentHeader: userAgentHeader,
	}, nil
}

// Shutdown stops the exporter from accepting incoming calls(and return error), and wait for current export operations
// to finish before returning
func (prwe *PrwExporter) Shutdown(context.Context) error {
	close(prwe.closeChan)
	prwe.wg.Wait()
	return nil
}

// PushMetrics converts metrics to Prometheus remote write TimeSeries and send to remote endpoint. It maintain a map of
// TimeSeries, validates and handles each individual metric, adding the converted TimeSeries to the map, and finally
// exports the map.
func (prwe *PrwExporter) PushMetrics(ctx context.Context, md pdata.Metrics) error {
	prwe.wg.Add(1)
	defer prwe.wg.Done()

	select {
	case <-prwe.closeChan:
		return errors.New("shutdown has been called")
	default:
		tsMap := map[string]*prompb.TimeSeries{}
		dropped := 0
		var errs []error
		resourceMetricsSlice := md.ResourceMetrics()
		for i := 0; i < resourceMetricsSlice.Len(); i++ {
			resourceMetrics := resourceMetricsSlice.At(i)
			resource := resourceMetrics.Resource()
			instrumentationLibraryMetricsSlice := resourceMetrics.InstrumentationLibraryMetrics()
			// TODO: add resource attributes as labels, probably in next PR
			for j := 0; j < instrumentationLibraryMetricsSlice.Len(); j++ {
				instrumentationLibraryMetrics := instrumentationLibraryMetricsSlice.At(j)
				metricSlice := instrumentationLibraryMetrics.Metrics()

				// TODO: decide if instrumentation library information should be exported as labels
				for k := 0; k < metricSlice.Len(); k++ {
					metric := metricSlice.At(k)

					// check for valid type and temporality combination and for matching data field and type
					if ok := validateMetrics(metric); !ok {
						dropped++
						errs = append(errs, consumererror.Permanent(errors.New("invalid temporality and type combination")))
						continue
					}

					// handle individual metric based on type
					switch metric.DataType() {
					case pdata.MetricDataTypeDoubleSum, pdata.MetricDataTypeIntSum, pdata.MetricDataTypeDoubleGauge, pdata.MetricDataTypeIntGauge:
						if err := prwe.handleScalarMetric(tsMap, resource, metric); err != nil {
							dropped++
							errs = append(errs, consumererror.Permanent(err))
						}
					case pdata.MetricDataTypeHistogram, pdata.MetricDataTypeIntHistogram:
						if err := prwe.handleHistogramMetric(tsMap, resource, metric); err != nil {
							dropped++
							errs = append(errs, consumererror.Permanent(err))
						}
					case pdata.MetricDataTypeSummary:
						if err := prwe.handleSummaryMetric(tsMap, resource, metric); err != nil {
							dropped++
							errs = append(errs, consumererror.Permanent(err))
						}
					default:
						dropped++
						errs = append(errs, consumererror.Permanent(errors.New("unsupported metric type")))
					}
				}
			}
		}

		if exportErrors := prwe.export(ctx, tsMap); len(exportErrors) != 0 {
			dropped = md.MetricCount()
			errs = append(errs, exportErrors...)
		}

		if dropped != 0 {
			return consumererror.Combine(errs)
		}

		return nil
	}
}

func validateAndSanitizeExternalLabels(externalLabels map[string]string) (map[string]string, error) {
	sanitizedLabels := make(map[string]string)
	for key, value := range externalLabels {
		if key == "" || value == "" {
			return nil, fmt.Errorf("prometheus remote write: external labels configuration contains an empty key or value")
		}

		// Sanitize label keys to meet Prometheus Requirements
		if len(key) > 2 && key[:2] == "__" {
			key = "__" + sanitize(key[2:])
		} else {
			key = sanitize(key)
		}
		sanitizedLabels[key] = value
	}

	return sanitizedLabels, nil
}

// handleScalarMetric processes data points in a single OTLP scalar metric by adding the each point as a Sample into
// its corresponding TimeSeries in tsMap.
// tsMap and metric cannot be nil, and metric must have a non-nil descriptor
func (prwe *PrwExporter) handleScalarMetric(tsMap map[string]*prompb.TimeSeries, resource pdata.Resource, metric pdata.Metric) error {
	switch metric.DataType() {
	// int points
	case pdata.MetricDataTypeDoubleGauge:
		dataPoints := metric.DoubleGauge().DataPoints()
		if dataPoints.Len() == 0 {
			return fmt.Errorf("empty data points. %s is dropped", metric.Name())
		}

		for i := 0; i < dataPoints.Len(); i++ {
			addSingleDoubleDataPoint(dataPoints.At(i), resource, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	case pdata.MetricDataTypeIntGauge:
		dataPoints := metric.IntGauge().DataPoints()
		if dataPoints.Len() == 0 {
			return fmt.Errorf("empty data points. %s is dropped", metric.Name())
		}
		for i := 0; i < dataPoints.Len(); i++ {
			addSingleIntDataPoint(dataPoints.At(i), resource, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	case pdata.MetricDataTypeDoubleSum:
		dataPoints := metric.DoubleSum().DataPoints()
		if dataPoints.Len() == 0 {
			return fmt.Errorf("empty data points. %s is dropped", metric.Name())
		}
		for i := 0; i < dataPoints.Len(); i++ {
			addSingleDoubleDataPoint(dataPoints.At(i), resource, metric, prwe.namespace, tsMap, prwe.externalLabels)

		}
	case pdata.MetricDataTypeIntSum:
		dataPoints := metric.IntSum().DataPoints()
		if dataPoints.Len() == 0 {
			return fmt.Errorf("empty data points. %s is dropped", metric.Name())
		}
		for i := 0; i < dataPoints.Len(); i++ {
			addSingleIntDataPoint(dataPoints.At(i), resource, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	}
	return nil
}

// handleHistogramMetric processes data points in a single OTLP histogram metric by mapping the sum, count and each
// bucket of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *PrwExporter) handleHistogramMetric(tsMap map[string]*prompb.TimeSeries, resource pdata.Resource, metric pdata.Metric) error {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntHistogram:
		dataPoints := metric.IntHistogram().DataPoints()
		if dataPoints.Len() == 0 {
			return fmt.Errorf("empty data points. %s is dropped", metric.Name())
		}
		for i := 0; i < dataPoints.Len(); i++ {
			addSingleIntHistogramDataPoint(dataPoints.At(i), resource, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	case pdata.MetricDataTypeHistogram:
		dataPoints := metric.Histogram().DataPoints()
		if dataPoints.Len() == 0 {
			return fmt.Errorf("empty data points. %s is dropped", metric.Name())
		}
		for i := 0; i < dataPoints.Len(); i++ {
			addSingleDoubleHistogramDataPoint(dataPoints.At(i), resource, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	}
	return nil
}

// handleSummaryMetric processes data points in a single OTLP summary metric by mapping the sum, count and each
// quantile of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *PrwExporter) handleSummaryMetric(tsMap map[string]*prompb.TimeSeries, resource pdata.Resource, metric pdata.Metric) error {
	dataPoints := metric.Summary().DataPoints()
	if dataPoints.Len() == 0 {
		return fmt.Errorf("empty data points. %s is dropped", metric.Name())
	}
	for i := 0; i < dataPoints.Len(); i++ {
		addSingleDoubleSummaryDataPoint(dataPoints.At(i), resource, metric, prwe.namespace, tsMap, prwe.externalLabels)
	}
	return nil
}

// export sends a Snappy-compressed WriteRequest containing TimeSeries to a remote write endpoint in order
func (prwe *PrwExporter) export(ctx context.Context, tsMap map[string]*prompb.TimeSeries) []error {
	var errs []error
	// Calls the helper function to convert and batch the TsMap to the desired format
	requests, err := batchTimeSeries(tsMap, maxBatchByteSize)
	if err != nil {
		errs = append(errs, consumererror.Permanent(err))
		return errs
	}

	input := make(chan *prompb.WriteRequest, len(requests))
	for _, request := range requests {
		input <- request
	}
	close(input)

	var mu sync.Mutex
	var wg sync.WaitGroup

	concurrencyLimit := int(math.Min(maxConcurrentRequests, float64(len(requests))))
	wg.Add(concurrencyLimit) // used to wait for workers to be finished

	// Run concurrencyLimit of workers until there
	// is no more requests to execute in the input channel.
	for i := 0; i < concurrencyLimit; i++ {
		go func() {
			defer wg.Done()

			for request := range input {
				err := prwe.execute(ctx, request)
				if err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	return errs
}

func (prwe *PrwExporter) execute(ctx context.Context, writeReq *prompb.WriteRequest) error {
	// Uses proto.Marshal to convert the WriteRequest into bytes array
	data, err := proto.Marshal(writeReq)
	if err != nil {
		return consumererror.Permanent(err)
	}
	buf := make([]byte, len(data), cap(data))
	compressedData := snappy.Encode(buf, data)

	// Create the HTTP POST request to send to the endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", prwe.endpointURL.String(), bytes.NewReader(compressedData))
	if err != nil {
		return consumererror.Permanent(err)
	}

	// Add necessary headers specified by:
	// https://cortexmetrics.io/docs/apis/#remote-api
	req.Header.Add("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.Header.Set("User-Agent", prwe.userAgentHeader)

	resp, err := prwe.client.Do(req)
	if err != nil {
		return consumererror.Permanent(err)
	}
	defer resp.Body.Close()

	// 2xx status code is considered a success
	// 5xx errors are recoverable and the exporter should retry
	// Reference for different behavior according to status code:
	// https://github.com/prometheus/prometheus/pull/2552/files#diff-ae8db9d16d8057358e49d694522e7186
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 256))
	rerr := fmt.Errorf("remote write returned HTTP status %v; err = %v: %s", resp.Status, err, body)
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		return rerr
	}
	return consumererror.Permanent(rerr)
}
