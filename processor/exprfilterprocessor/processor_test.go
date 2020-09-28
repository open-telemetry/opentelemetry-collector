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

package labelfilterprocessor

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/goldendataset"
)

const filteredMetric = "p0_metric_1"
const filteredLblKey = "pt-label-key-1"
const filteredLblVal = "pt-label-val-1"

func TestProcessor(t *testing.T) {
	filtered := filterMetrics(t, q(), 2)
	for _, metrics := range filtered {
		rmsSlice := metrics.ResourceMetrics()
		for i := 0; i < rmsSlice.Len(); i++ {
			rms := rmsSlice.At(i)
			ilms := rms.InstrumentationLibraryMetrics()
			for j := 0; j < ilms.Len(); j++ {
				ilm := ilms.At(j)
				metricSlice := ilm.Metrics()
				for k := 0; k < metricSlice.Len(); k++ {
					metric := metricSlice.At(k)
					if metric.Name() == filteredMetric {
						dt := metric.DataType()
						switch dt {
						case pdata.MetricDataTypeIntGauge:
							pts := metric.IntGauge().DataPoints()
							for l := 0; l < pts.Len(); l++ {
								pt := pts.At(l)
								pt.LabelsMap().ForEach(func(k string, v pdata.StringValue) {
									if k == filteredLblKey && v.Value() == filteredLblVal {
										require.Fail(t, "k == filteredLblKey && v.Value() == filteredLblVal")
									}
								})
							}
						case pdata.MetricDataTypeDoubleGauge:
						case pdata.MetricDataTypeIntSum:
						case pdata.MetricDataTypeDoubleSum:
						case pdata.MetricDataTypeIntHistogram:
						case pdata.MetricDataTypeDoubleHistogram:
						}
					}
				}
			}
		}
	}
}

func filterMetrics(t *testing.T, q string, size int) []pdata.Metrics {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	pCfg := cfg.(*Config)
	pCfg.Exclude = []string{q}

	ctx := context.Background()
	next := &exportertest.SinkMetricsExporter{}
	proc, err := factory.CreateMetricsProcessor(
		ctx,
		component.ProcessorCreateParams{},
		cfg,
		next,
	)
	require.NoError(t, err)
	require.NotNil(t, proc)

	capabilities := proc.GetCapabilities()
	assert.False(t, capabilities.MutatesConsumedData)

	mds := testDataSlice(size)

	inputCount := 0
	for _, md := range mds {
		inputCount += md.MetricCount()
		err = proc.ConsumeMetrics(ctx, md)
		require.NoError(t, err)
	}
	require.Equal(t, inputCount-1, next.MetricsCount())
	return next.AllMetrics()
}

func BenchmarkProcessor1(b *testing.B) {
	benchmarkProcessor(b, 1)
}

func BenchmarkProcessor2(b *testing.B) {
	benchmarkProcessor(b, 2)
}

func BenchmarkProcessor4(b *testing.B) {
	benchmarkProcessor(b, 4)
}

func benchmarkProcessor(b *testing.B, size int) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	pCfg := cfg.(*Config)
	pCfg.Exclude = []string{q()}
	ctx := context.Background()
	nextConsumer := &exportertest.SinkMetricsExporter{}
	proc, _ := factory.CreateMetricsProcessor(
		ctx,
		component.ProcessorCreateParams{},
		cfg,
		nextConsumer,
	)

	mds := testDataSlice(size)
	for i := 0; i < b.N; i++ {
		for _, md := range mds {
			_ = proc.ConsumeMetrics(ctx, md)
		}
	}
}

func testDataSlice(size int) []pdata.Metrics {
	var out []pdata.Metrics
	for i := 0; i < 16; i++ {
		out = append(out, testData(fmt.Sprintf("p%d_", i), size))
	}
	return out
}

func testData(prefix string, size int) pdata.Metrics {
	c := goldendataset.MetricCfg{
		MetricDescriptorType: pdata.MetricDataTypeIntGauge,
		MetricNamePrefix:     prefix,
		NumILMPerResource:    size,
		NumMetricsPerILM:     size,
		NumPtLabels:          size,
		NumPtsPerMetric:      size,
		NumResourceAttrs:     size,
		NumResourceMetrics:   size,
	}
	return goldendataset.MetricDataFromCfg(c)
}

func q() string {
	format := "MetricName == '%s' && LabelsMap['%s'] == '%s'"
	return fmt.Sprintf(format, filteredMetric, filteredLblKey, filteredLblVal)
}
