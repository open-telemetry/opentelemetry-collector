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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
)

func TestRetries(t *testing.T) {
	var retries int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if retries == 3 {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(500) // retryable
		retries++
	}))
	defer server.Close()

	defaultConfig := createDefaultConfig().(*Config)
	defaultConfig.RemoteWriteQueue.MinBackoff = 1 * time.Millisecond
	defaultConfig.RemoteWriteQueue.MaxBackoff = 50 * time.Millisecond
	buildInfo := component.BuildInfo{
		Description: "OpenTelemetry Collector",
		Version:     "1.0",
	}
	exporter, err := NewPrwExporter(defaultConfig, server.URL, http.DefaultClient, buildInfo)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", server.URL, nil)
	assert.NoError(t, exporter.makeReqWithRetries(req))
	assert.Equal(t, retries, 3)
}
