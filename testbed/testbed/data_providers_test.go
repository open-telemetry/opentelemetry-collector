// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testbed

import (
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"go.opentelemetry.io/collector/internal/data"
)

const metricsPictPairsFile = "../../internal/goldendataset/testdata/generated_pict_pairs_metrics.txt"

func TestGenerateMetrics(t *testing.T) {
	dp := NewGoldenDataProvider("", "", metricsPictPairsFile, 42)
	dp.SetLoadGeneratorCounters(atomic.NewUint64(0), atomic.NewUint64(0))
	var ms []data.MetricData
	for true {
		m, done := dp.GenerateMetrics()
		if done {
			break
		}
		ms = append(ms, m)
	}
	require.Equal(t, len(dp.metricsGenerated), len(ms))
}

func TestGenerateMetricsOld(t *testing.T) {
	dp := NewGoldenDataProvider("", "", metricsPictPairsFile, 42)
	dp.SetLoadGeneratorCounters(atomic.NewUint64(0), atomic.NewUint64(0))
	var ms [][]*metricspb.Metric
	for {
		m, done := dp.GenerateMetricsOld()
		if done {
			break
		}
		ms = append(ms, m)
	}
	require.Equal(t, len(dp.metricsGenerated), len(ms))
}
