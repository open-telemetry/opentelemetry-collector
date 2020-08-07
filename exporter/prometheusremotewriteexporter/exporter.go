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

// This file defines the prwExporter struct and its class functions.
// The exporter is in charge of receiving data upstream from the Collector pipeline,
// Converting the data to Prometheus TimeSeries data, and then sending to the configurable
// Remote Write HTTP Endpoint.

package prometheusremotewriteexporter

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/pkg/errors"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
)

// prwExporter converts OTLP metrics to Prometheus remote write TimeSeries and sends them to a remote endpoint
type prwExporter struct {
	namespace   string
	endpointURL *url.URL
	client      *http.Client
	headers     map[string]string
	wg          *sync.WaitGroup
	closeChan   chan struct{}
}

// newPrwExporter initializes a new prwExporter instance and sets fields accordingly.
// client parameter cannot be nil.
func newPrwExporter(namespace string, endpoint string, client *http.Client) (*prwExporter, error) {

	if namespace == "" {
		return nil, errors.Errorf("namespace cannot be empty")
	}

	if client == nil {
		return nil, errors.Errorf("http client cannot be nil")
	}

	endpointURL, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, errors.Errorf("invalid endpoint")
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
		return pdatautil.MetricCount(md), errors.Errorf("shutdown has been called")
	default:
		return 0, nil

		/*
			Actual code is implemented in the next PR. The pushMetrics method will
			create a string-TimeSeries map, convert the inputted OTLP metrics to the appropriate
			TimeSeries equivalent, add the TimeSeries to the map, and then send the map to the HTTP Endpoint
			via Export() which will also be added in the next PR.
		*/
	}
}
