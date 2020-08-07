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

package prometheusremotewriteexporter

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

// Test_newPrwExporter checks that a new exporter instance with non-nil fields is initialized
func Test_newPrwExporter(t *testing.T) {
	config := &Config{
		ExporterSettings:   configmodels.ExporterSettings{},
		TimeoutSettings:    exporterhelper.TimeoutSettings{},
		QueueSettings:      exporterhelper.QueueSettings{},
		RetrySettings:      exporterhelper.RetrySettings{},
		Namespace:          "",
		HTTPClientSettings: confighttp.HTTPClientSettings{Endpoint: ""},
	}
	tests := []struct {
		name        string
		config      *Config
		namespace   string
		endpoint    string
		client      *http.Client
		returnError bool
	}{
		{
			"invalid_URL",
			config,
			"test",
			"invalid URL",
			http.DefaultClient,
			true,
		},
		{
			"nil_client",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			nil,
			true,
		},
		{
			"success_case",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			http.DefaultClient,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prwe, err := newPrwExporter(tt.namespace, tt.endpoint, tt.client)
			if tt.returnError {
				assert.Error(t, err)
				return
			}
			require.NotNil(t, prwe)
			assert.NotNil(t, prwe.namespace)
			assert.NotNil(t, prwe.endpointURL)
			assert.NotNil(t, prwe.client)
			assert.NotNil(t, prwe.closeChan)
			assert.NotNil(t, prwe.wg)
		})
	}
}

// Test_shutdown checks after shutdown is called, incoming calls to pushMetrics return error.
func Test_shutdown(t *testing.T) {
	prwe := &prwExporter{
		wg:        new(sync.WaitGroup),
		closeChan: make(chan struct{}),
	}
	wg := new(sync.WaitGroup)
	errChan := make(chan error, 5)
	err := prwe.shutdown(context.Background())
	require.NoError(t, err)
	errChan = make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := prwe.pushMetrics(context.Background(),
				pdatautil.MetricsFromInternalMetrics(testdata.GenerateMetricDataEmpty()))
			errChan <- ok
		}()
	}
	wg.Wait()
	close(errChan)
	for ok := range errChan {
		assert.Error(t, ok)
	}
}
