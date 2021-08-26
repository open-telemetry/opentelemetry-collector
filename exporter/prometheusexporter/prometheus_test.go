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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestPrometheusExporter(t *testing.T) {
	tests := []struct {
		config       *Config
		wantErr      string
		wantStartErr string
	}{
		{
			config: &Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				Namespace:        "test",
				ConstLabels: map[string]string{
					"foo0":  "bar0",
					"code0": "one0",
				},
				Endpoint:         ":8999",
				SendTimestamps:   false,
				MetricExpiration: 60 * time.Second,
			},
		},
		{
			config: &Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				Endpoint:         ":88999",
			},
			wantStartErr: "listen tcp: address 88999: invalid port",
		},
		{
			config: &Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
			},
			wantErr: "expecting a non-blank address to run the Prometheus metrics handler",
		},
	}

	factory := NewFactory()
	set := componenttest.NewNopExporterCreateSettings()
	for _, tt := range tests {
		// Run it a few times to ensure that shutdowns exit cleanly.
		for j := 0; j < 3; j++ {
			exp, err := factory.CreateMetricsExporter(context.Background(), set, tt.config)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				continue
			} else {
				require.NoError(t, err)
			}

			assert.NotNil(t, exp)
			err = exp.Start(context.Background(), componenttest.NewNopHost())

			if tt.wantStartErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantStartErr, err.Error())
			} else {
				require.NoError(t, err)
			}

			require.NoError(t, exp.Shutdown(context.Background()))
		}
	}
}

func TestPrometheusExporter_endToEnd(t *testing.T) {
	cfg := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
		Namespace:        "test",
		ConstLabels: map[string]string{
			"foo1":  "bar1",
			"code1": "one1",
		},
		Endpoint:         ":7777",
		MetricExpiration: 120 * time.Minute,
	}

	factory := NewFactory()
	set := componenttest.NewNopExporterCreateSettings()
	exp, err := factory.CreateMetricsExporter(context.Background(), set, cfg)
	assert.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
		// trigger a get so that the server cleans up our keepalive socket
		_, err = http.Get("http://localhost:7777/metrics")
		require.NoError(t, err)
	})

	assert.NotNil(t, exp)

	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))

	// Should accumulate multiple metrics
	assert.NoError(t, exp.ConsumeMetrics(context.Background(), metricBuilder(128, "metric_1_")))

	for delta := 0; delta <= 20; delta += 10 {
		assert.NoError(t, exp.ConsumeMetrics(context.Background(), metricBuilder(int64(delta), "metric_2_")))

		res, err1 := http.Get("http://localhost:7777/metrics")
		require.NoError(t, err1, "Failed to perform a scrape")

		if g, w := res.StatusCode, 200; g != w {
			t.Errorf("Mismatched HTTP response status code: Got: %d Want: %d", g, w)
		}
		blob, _ := ioutil.ReadAll(res.Body)
		_ = res.Body.Close()
		want := []string{
			`# HELP test_metric_1_this_one_there_where_ Extra ones`,
			`# TYPE test_metric_1_this_one_there_where_ counter`,
			fmt.Sprintf(`test_metric_1_this_one_there_where_{arch="x86",code1="one1",foo1="bar1",os="windows"} %v`, 99+128),
			fmt.Sprintf(`test_metric_1_this_one_there_where_{arch="x86",code1="one1",foo1="bar1",os="linux"} %v`, 100+128),
			`# HELP test_metric_2_this_one_there_where_ Extra ones`,
			`# TYPE test_metric_2_this_one_there_where_ counter`,
			fmt.Sprintf(`test_metric_2_this_one_there_where_{arch="x86",code1="one1",foo1="bar1",os="windows"} %v`, 99+delta),
			fmt.Sprintf(`test_metric_2_this_one_there_where_{arch="x86",code1="one1",foo1="bar1",os="linux"} %v`, 100+delta),
		}

		for _, w := range want {
			if !strings.Contains(string(blob), w) {
				t.Errorf("Missing %v from response:\n%v", w, string(blob))
			}
		}
	}

	// Expired metrics should be removed during first scrape
	exp.(*wrapMetricsExpoter).exporter.collector.accumulator.(*lastValueAccumulator).metricExpiration = 1 * time.Millisecond
	time.Sleep(10 * time.Millisecond)

	res, err := http.Get("http://localhost:7777/metrics")
	require.NoError(t, err, "Failed to perform a scrape")

	if g, w := res.StatusCode, 200; g != w {
		t.Errorf("Mismatched HTTP response status code: Got: %d Want: %d", g, w)
	}
	blob, _ := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	require.Emptyf(t, string(blob), "Metrics did not expire")
}

func TestPrometheusExporter_endToEndWithTimestamps(t *testing.T) {
	cfg := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
		Namespace:        "test",
		ConstLabels: map[string]string{
			"foo2":  "bar2",
			"code2": "one2",
		},
		Endpoint:         ":7777",
		SendTimestamps:   true,
		MetricExpiration: 120 * time.Minute,
	}

	factory := NewFactory()
	set := componenttest.NewNopExporterCreateSettings()
	exp, err := factory.CreateMetricsExporter(context.Background(), set, cfg)
	assert.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
		// trigger a get so that the server cleans up our keepalive socket
		_, err = http.Get("http://localhost:7777/metrics")
		require.NoError(t, err)
	})

	assert.NotNil(t, exp)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))

	// Should accumulate multiple metrics

	assert.NoError(t, exp.ConsumeMetrics(context.Background(), metricBuilder(128, "metric_1_")))

	for delta := 0; delta <= 20; delta += 10 {
		assert.NoError(t, exp.ConsumeMetrics(context.Background(), metricBuilder(int64(delta), "metric_2_")))

		res, err1 := http.Get("http://localhost:7777/metrics")
		require.NoError(t, err1, "Failed to perform a scrape")

		if g, w := res.StatusCode, 200; g != w {
			t.Errorf("Mismatched HTTP response status code: Got: %d Want: %d", g, w)
		}
		blob, _ := ioutil.ReadAll(res.Body)
		_ = res.Body.Close()
		want := []string{
			`# HELP test_metric_1_this_one_there_where_ Extra ones`,
			`# TYPE test_metric_1_this_one_there_where_ counter`,
			fmt.Sprintf(`test_metric_1_this_one_there_where_{arch="x86",code2="one2",foo2="bar2",os="windows"} %v %v`, 99+128, 1543160298100+128000),
			fmt.Sprintf(`test_metric_1_this_one_there_where_{arch="x86",code2="one2",foo2="bar2",os="linux"} %v %v`, 100+128, 1543160298100),
			`# HELP test_metric_2_this_one_there_where_ Extra ones`,
			`# TYPE test_metric_2_this_one_there_where_ counter`,
			fmt.Sprintf(`test_metric_2_this_one_there_where_{arch="x86",code2="one2",foo2="bar2",os="windows"} %v %v`, 99+delta, 1543160298100+delta*1000),
			fmt.Sprintf(`test_metric_2_this_one_there_where_{arch="x86",code2="one2",foo2="bar2",os="linux"} %v %v`, 100+delta, 1543160298100),
		}

		for _, w := range want {
			if !strings.Contains(string(blob), w) {
				t.Errorf("Missing %v from response:\n%v", w, string(blob))
			}
		}
	}

	// Expired metrics should be removed during first scrape
	exp.(*wrapMetricsExpoter).exporter.collector.accumulator.(*lastValueAccumulator).metricExpiration = 1 * time.Millisecond
	time.Sleep(10 * time.Millisecond)

	res, err := http.Get("http://localhost:7777/metrics")
	require.NoError(t, err, "Failed to perform a scrape")

	if g, w := res.StatusCode, 200; g != w {
		t.Errorf("Mismatched HTTP response status code: Got: %d Want: %d", g, w)
	}
	blob, _ := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	require.Emptyf(t, string(blob), "Metrics did not expire")
}

func TestPrometheusExporter_endToEndWithResource(t *testing.T) {
	cfg := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
		Namespace:        "test",
		ConstLabels: map[string]string{
			"foo2":  "bar2",
			"code2": "one2",
		},
		Endpoint:         ":7777",
		SendTimestamps:   true,
		MetricExpiration: 120 * time.Minute,
		ResourceToTelemetrySettings: exporterhelper.ResourceToTelemetrySettings{
			Enabled: true,
		},
	}

	factory := NewFactory()
	set := componenttest.NewNopExporterCreateSettings()
	exp, err := factory.CreateMetricsExporter(context.Background(), set, cfg)
	assert.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
		// trigger a get so that the server cleans up our keepalive socket
		http.Get("http://localhost:7777/metrics")
	})

	assert.NotNil(t, exp)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))

	md := testdata.GenerateMetricsOneMetric()
	assert.NotNil(t, md)

	assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))

	rsp, err := http.Get("http://localhost:7777/metrics")
	require.NoError(t, err, "Failed to perform a scrape")

	if g, w := rsp.StatusCode, 200; g != w {
		t.Errorf("Mismatched HTTP response status code: Got: %d Want: %d", g, w)
	}

	blob, _ := ioutil.ReadAll(rsp.Body)
	_ = rsp.Body.Close()

	want := []string{
		`# HELP test_counter_int`,
		`# TYPE test_counter_int counter`,
		`test_counter_int{code2="one2",foo2="bar2",label_1="label-value-1",resource_attr="resource-attr-val-1"} 123 1581452773000`,
		`test_counter_int{code2="one2",foo2="bar2",label_2="label-value-2",resource_attr="resource-attr-val-1"} 456 1581452773000`,
	}

	for _, w := range want {
		if !strings.Contains(string(blob), w) {
			t.Errorf("Missing %v from response:\n%v", w, string(blob))
		}
	}
}

func metricBuilder(delta int64, prefix string) pdata.Metrics {
	md := pdata.NewMetrics()
	ms := md.ResourceMetrics().AppendEmpty().InstrumentationLibraryMetrics().AppendEmpty().Metrics()

	m1 := ms.AppendEmpty()
	m1.SetName(prefix + "this/one/there(where)")
	m1.SetDescription("Extra ones")
	m1.SetUnit("1")
	m1.SetDataType(pdata.MetricDataTypeSum)
	d1 := m1.Sum()
	d1.SetIsMonotonic(true)
	d1.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp1 := d1.DataPoints().AppendEmpty()
	dp1.SetStartTimestamp(pdata.TimestampFromTime(time.Unix(1543160298+delta, 100000090)))
	dp1.SetTimestamp(pdata.TimestampFromTime(time.Unix(1543160298+delta, 100000997)))
	dp1.Attributes().UpsertString("os", "windows")
	dp1.Attributes().UpsertString("arch", "x86")
	dp1.SetIntVal(99 + delta)

	m2 := ms.AppendEmpty()
	m2.SetName(prefix + "this/one/there(where)")
	m2.SetDescription("Extra ones")
	m2.SetUnit("1")
	m2.SetDataType(pdata.MetricDataTypeSum)
	d2 := m2.Sum()
	d2.SetIsMonotonic(true)
	d2.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp2 := d2.DataPoints().AppendEmpty()
	dp2.SetStartTimestamp(pdata.TimestampFromTime(time.Unix(1543160298, 100000090)))
	dp2.SetTimestamp(pdata.TimestampFromTime(time.Unix(1543160298, 100000997)))
	dp2.Attributes().UpsertString("os", "linux")
	dp2.Attributes().UpsertString("arch", "x86")
	dp2.SetIntVal(100 + delta)

	return md
}
