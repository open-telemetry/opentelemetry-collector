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

// Note: implementation for this class is in a separate PR
package prometheusremotewriteexporter

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/pkg/errors"

	"go.opentelemetry.io/collector/consumer/pdata"
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
	return 0, nil
}
