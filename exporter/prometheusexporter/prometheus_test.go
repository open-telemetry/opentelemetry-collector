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

package prometheusexporter

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func TestPrometheusExporter(t *testing.T) {
	tests := []struct {
		config  *Config
		wantErr string
	}{
		{
			config: &Config{
				Namespace: "test",
				ConstLabels: map[string]string{
					"foo0":  "bar0",
					"code0": "one0",
				},
				Endpoint:       ":8999",
				SendTimestamps: false,
			},
		},
		{
			config:  &Config{},
			wantErr: "expecting a non-blank address to run the Prometheus metrics handler",
		},
	}

	factory := NewFactory()
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	for _, tt := range tests {
		// Run it a few times to ensure that shutdowns exit cleanly.
		for j := 0; j < 3; j++ {
			exp, err := factory.CreateMetricsExporter(context.Background(), creationParams, tt.config)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				continue
			} else {
				require.NoError(t, err)
			}

			assert.NotNil(t, exp)
			require.Nil(t, err)
			require.NoError(t, exp.Shutdown(context.Background()))
		}
	}
}

func TestPrometheusExporter_endToEnd(t *testing.T) {
	config := &Config{
		Namespace: "test",
		ConstLabels: map[string]string{
			"foo1":  "bar1",
			"code1": "one1",
		},
		Endpoint: ":7777",
	}

	factory := NewFactory()
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateMetricsExporter(context.Background(), creationParams, config)
	assert.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
		// trigger a get so that the server cleans up our keepalive socket
		http.Get("http://localhost:7777/metrics")
	})

	assert.NotNil(t, exp)

	for delta := 0; delta <= 20; delta += 10 {
		md := internaldata.OCToMetrics(consumerdata.MetricsData{Metrics: metricBuilder(int64(delta))})
		assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))

		res, err := http.Get("http://localhost:7777/metrics")
		require.NoError(t, err, "Failed to perform a scrape")

		if g, w := res.StatusCode, 200; g != w {
			t.Errorf("Mismatched HTTP response status code: Got: %d Want: %d", g, w)
		}
		blob, _ := ioutil.ReadAll(res.Body)
		_ = res.Body.Close()
		want := []string{
			`# HELP test_this_one_there_where_ Extra ones`,
			`# TYPE test_this_one_there_where_ counter`,
			fmt.Sprintf(`test_this_one_there_where_{arch="x86",code1="one1",foo1="bar1",os="windows"} %v`, 99+delta),
			fmt.Sprintf(`test_this_one_there_where_{arch="x86",code1="one1",foo1="bar1",os="linux"} %v`, 100+delta),
		}

		for _, w := range want {
			if !strings.Contains(string(blob), w) {
				t.Errorf("Missing %v from response:\n%v", w, string(blob))
			}
		}
	}
}

func TestPrometheusExporter_endToEndWithTimestamps(t *testing.T) {
	config := &Config{
		Namespace: "test",
		ConstLabels: map[string]string{
			"foo2":  "bar2",
			"code2": "one2",
		},
		Endpoint:       ":7777",
		SendTimestamps: true,
	}

	factory := NewFactory()
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateMetricsExporter(context.Background(), creationParams, config)
	assert.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
		// trigger a get so that the server cleans up our keepalive socket
		http.Get("http://localhost:7777/metrics")
	})

	assert.NotNil(t, exp)

	for delta := 0; delta <= 20; delta += 10 {
		md := internaldata.OCToMetrics(consumerdata.MetricsData{Metrics: metricBuilder(int64(delta))})
		assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))

		res, err := http.Get("http://localhost:7777/metrics")
		require.NoError(t, err, "Failed to perform a scrape")

		if g, w := res.StatusCode, 200; g != w {
			t.Errorf("Mismatched HTTP response status code: Got: %d Want: %d", g, w)
		}
		blob, _ := ioutil.ReadAll(res.Body)
		_ = res.Body.Close()
		want := []string{
			`# HELP test_this_one_there_where_ Extra ones`,
			`# TYPE test_this_one_there_where_ counter`,
			fmt.Sprintf(`test_this_one_there_where_{arch="x86",code2="one2",foo2="bar2",os="windows"} %v %v`, 99+delta, 1543160298100),
			fmt.Sprintf(`test_this_one_there_where_{arch="x86",code2="one2",foo2="bar2",os="linux"} %v %v`, 100+delta, 1543160298100),
		}

		for _, w := range want {
			if !strings.Contains(string(blob), w) {
				t.Errorf("Missing %v from response:\n%v", w, string(blob))
			}
		}
	}
}

func metricBuilder(delta int64) []*metricspb.Metric {
	return []*metricspb.Metric{
		{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        "this/one/there(where)",
				Description: "Extra ones",
				Unit:        "1",
				Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
				LabelKeys:   []*metricspb.LabelKey{{Key: "os"}, {Key: "arch"}},
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					StartTimestamp: &timestamppb.Timestamp{
						Seconds: 1543160298,
						Nanos:   100000090,
					},
					LabelValues: []*metricspb.LabelValue{
						{Value: "windows", HasValue: true},
						{Value: "x86", HasValue: true},
					},
					Points: []*metricspb.Point{
						{
							Timestamp: &timestamppb.Timestamp{
								Seconds: 1543160298,
								Nanos:   100000997,
							},
							Value: &metricspb.Point_Int64Value{
								Int64Value: 99 + delta,
							},
						},
					},
				},
			},
		},
		{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        "this/one/there(where)",
				Description: "Extra ones",
				Unit:        "1",
				Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
				LabelKeys:   []*metricspb.LabelKey{{Key: "os"}, {Key: "arch"}},
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					StartTimestamp: &timestamppb.Timestamp{
						Seconds: 1543160298,
						Nanos:   100000090,
					},
					LabelValues: []*metricspb.LabelValue{
						{Value: "linux", HasValue: true},
						{Value: "x86", HasValue: true},
					},
					Points: []*metricspb.Point{
						{
							Timestamp: &timestamppb.Timestamp{
								Seconds: 1543160298,
								Nanos:   100000997,
							},
							Value: &metricspb.Point_Int64Value{
								Int64Value: 100 + delta,
							},
						},
					},
				},
			},
		},
	}
}
