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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	"go.opentelemetry.io/collector/internal/version"
)

const (
	maxConcurrentRequests = 5
	maxBatchByteSize      = 3000000
)

// PrwExporter converts OTLP metrics to Prometheus remote write TimeSeries and sends them to a remote endpoint.
type PrwExporter struct {
	namespace      string
	externalLabels map[string]string
	endpointURL    *url.URL
	client         *http.Client
	wg             *sync.WaitGroup
	closeChan      chan struct{}
}

// NewPrwExporter initializes a new PrwExporter instance and sets fields accordingly.
// client parameter cannot be nil.
func NewPrwExporter(namespace string, endpoint string, client *http.Client, externalLabels map[string]string) (*PrwExporter, error) {
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

	return &PrwExporter{
		namespace:      namespace,
		externalLabels: sanitizedLabels,
		endpointURL:    endpointURL,
		client:         client,
		wg:             new(sync.WaitGroup),
		closeChan:      make(chan struct{}),
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
func (prwe *PrwExporter) PushMetrics(ctx context.Context, md pdata.Metrics) (int, error) {
	prwe.wg.Add(1)
	defer prwe.wg.Done()

	select {
	case <-prwe.closeChan:
		return md.MetricCount(), errors.New("shutdown has been called")
	default:
		tsMap := map[string]*prompb.TimeSeries{}
		dropped := 0
		var errs []error
		resourceMetrics := pdata.MetricsToOtlp(md)
		for _, resourceMetric := range resourceMetrics {
			if resourceMetric == nil {
				continue
			}
			// TODO: add resource attributes as labels, probably in next PR
			for _, instrumentationMetrics := range resourceMetric.InstrumentationLibraryMetrics {
				if instrumentationMetrics == nil {
					continue
				}
				// TODO: decide if instrumentation library information should be exported as labels
				for _, metric := range instrumentationMetrics.Metrics {
					if metric == nil {
						dropped++
						continue
					}
					// check for valid type and temporality combination and for matching data field and type
					if ok := validateMetrics(metric); !ok {
						dropped++
						errs = append(errs, consumererror.Permanent(errors.New("invalid temporality and type combination")))
						continue
					}
					// handle individual metric based on type
					switch metric.Data.(type) {
					case *otlp.Metric_DoubleSum, *otlp.Metric_IntSum, *otlp.Metric_DoubleGauge, *otlp.Metric_IntGauge:
						if err := prwe.handleScalarMetric(tsMap, metric); err != nil {
							dropped++
							errs = append(errs, consumererror.Permanent(err))
						}
					case *otlp.Metric_DoubleHistogram, *otlp.Metric_IntHistogram:
						if err := prwe.handleHistogramMetric(tsMap, metric); err != nil {
							dropped++
							errs = append(errs, consumererror.Permanent(err))
						}
					case *otlp.Metric_DoubleSummary:
						if err := prwe.handleSummaryMetric(tsMap, metric); err != nil {
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
			return dropped, componenterror.CombineErrors(errs)
		}

		return 0, nil
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
func (prwe *PrwExporter) handleScalarMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	switch metric.Data.(type) {
	// int points
	case *otlp.Metric_DoubleGauge:
		if metric.GetDoubleGauge().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetDoubleGauge().GetDataPoints() {
			addSingleDoubleDataPoint(pt, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	case *otlp.Metric_IntGauge:
		if metric.GetIntGauge().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetIntGauge().GetDataPoints() {
			addSingleIntDataPoint(pt, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	case *otlp.Metric_DoubleSum:
		if metric.GetDoubleSum().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetDoubleSum().GetDataPoints() {
			addSingleDoubleDataPoint(pt, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	case *otlp.Metric_IntSum:
		if metric.GetIntSum().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetIntSum().GetDataPoints() {
			addSingleIntDataPoint(pt, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	}
	return nil
}

// handleHistogramMetric processes data points in a single OTLP histogram metric by mapping the sum, count and each
// bucket of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *PrwExporter) handleHistogramMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	switch metric.Data.(type) {
	case *otlp.Metric_IntHistogram:
		if metric.GetIntHistogram().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetIntHistogram().GetDataPoints() {
			addSingleIntHistogramDataPoint(pt, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	case *otlp.Metric_DoubleHistogram:
		if metric.GetDoubleHistogram().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetDoubleHistogram().GetDataPoints() {
			addSingleDoubleHistogramDataPoint(pt, metric, prwe.namespace, tsMap, prwe.externalLabels)
		}
	}
	return nil
}

// handleSummaryMetric processes data points in a single OTLP summary metric by mapping the sum, count and each
// quantile of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *PrwExporter) handleSummaryMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	if metric.GetDoubleSummary().GetDataPoints() == nil {
		return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
	}
	for _, pt := range metric.GetDoubleSummary().GetDataPoints() {
		addSingleDoubleSummaryDataPoint(pt, metric, prwe.namespace, tsMap, prwe.externalLabels)
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
	req.Header.Set("User-Agent", "OpenTelemetry-Collector/"+version.Version)

	resp, err := prwe.client.Do(req)
	if err != nil {
		return consumererror.Permanent(err)
	}

	// 2xx status code is considered a success
	// 5xx errors are recoverable and the exporter should retry
	// Reference for different behavior according to status code:
	// https://github.com/prometheus/prometheus/pull/2552/files#diff-ae8db9d16d8057358e49d694522e7186
	if resp.StatusCode/100 != 2 {
		scanner := bufio.NewScanner(io.LimitReader(resp.Body, 256))
		var line string
		if scanner.Scan() {
			line = scanner.Text()
		}
		err := fmt.Errorf("server returned HTTP status %v: %v ", resp.Status, line)
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			return err
		}
		return consumererror.Permanent(err)
	}
	return nil
}
