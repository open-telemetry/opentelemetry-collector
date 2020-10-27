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

// Program otelcol is the OpenTelemetry Collector that collects stats
// and traces and exports to a configured backend.
package defaultcomponents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configmodels"
)

func TestDefaultComponents(t *testing.T) {
	expectedExtensions := []configmodels.Type{
		"health_check",
		"pprof",
		"zpages",
		"fluentbit",
	}
	expectedReceivers := []configmodels.Type{
		"jaeger",
		"zipkin",
		"prometheus",
		"opencensus",
		"otlp",
		"hostmetrics",
		"fluentforward",
		"kafka",
	}
	expectedProcessors := []configmodels.Type{
		"attributes",
		"resource",
		"queued_retry",
		"batch",
		"memory_limiter",
		"probabilistic_sampler",
		"span",
		"filter",
	}
	expectedExporters := []configmodels.Type{
		"opencensus",
		"prometheus",
		"prometheusremotewrite",
		"logging",
		"zipkin",
		"jaeger",
		"file",
		"otlp",
		"otlphttp",
		"kafka",
	}

	factories, err := Components()
	assert.NoError(t, err)

	exts := factories.Extensions
	assert.Equal(t, len(expectedExtensions), len(exts))
	for _, k := range expectedExtensions {
		v, ok := exts[k]
		assert.True(t, ok)
		assert.Equal(t, k, v.Type())
	}

	recvs := factories.Receivers
	assert.Equal(t, len(expectedReceivers), len(recvs))
	for _, k := range expectedReceivers {
		v, ok := recvs[k]
		require.True(t, ok)
		assert.Equal(t, k, v.Type())
		assert.Equal(t, k, v.CreateDefaultConfig().Type())
	}

	procs := factories.Processors
	assert.Equal(t, len(expectedProcessors), len(procs))
	for _, k := range expectedProcessors {
		v, ok := procs[k]
		require.True(t, ok)
		assert.Equal(t, k, v.Type())
		assert.Equal(t, k, v.CreateDefaultConfig().Type())
	}

	exps := factories.Exporters
	assert.Equal(t, len(expectedExporters), len(exps))
	for _, k := range expectedExporters {
		v, ok := exps[k]
		require.True(t, ok)
		assert.Equal(t, k, v.Type())
		assert.Equal(t, k, v.CreateDefaultConfig().Type())
	}
}
