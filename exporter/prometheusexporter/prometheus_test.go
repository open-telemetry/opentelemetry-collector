// Copyright 2019, OpenCensus Authors
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

package prometheusexporter

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	viperutils "github.com/census-instrumentation/opencensus-service/internal/config/viperutils"
)

func TestPrometheusExporter(t *testing.T) {
	tests := []struct {
		config  string
		wantErr string
	}{
		{
			config: `
prometheus:
    namespace: "test"
    const_labels: {
        "foo": "bar",
        "code": "one"
    }
    address: ":8999"
`,
		},
	}

	for i, tt := range tests {
		// Run it a few times to ensure that shutdowns exit cleanly.
		for j := 0; j < 3; j++ {
			v, _ := viperutils.ViperFromYAMLBytes([]byte(tt.config))
			tes, mes, doneFns, err := PrometheusExportersFromViper(v)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("#%d iteration #%d: Unexpected error: %v Wanted: %v", i, j, err, tt.wantErr)
				}
				continue
			}

			if err != nil {
				t.Errorf("#%d iteration #%d: unexpected parse error: %v", i, j, err)
			}
			if len(tes) != 0 {
				t.Errorf("#%d iteration #%d: Unexpectedly got back %d > 0 trace exporters", i, j, len(tes))
			}
			if len(mes) == 0 {
				t.Errorf("#%d iteration #%d: Unexpectedly got back 0 metrics exporters", i, j)
			}
			for _, doneFn := range doneFns {
				doneFn()
			}
		}
	}
}

func TestPrometheusExporter_nilDoesntCauseCrash(t *testing.T) {
	config := []byte(`
prometheus:`)
	v, _ := viperutils.ViperFromYAMLBytes([]byte(config))
	tes, mes, doneFns, err := PrometheusExportersFromViper(v)
	if err != nil {
		t.Errorf("Unexpected parse error: %v", err)
	}
	if len(tes) != 0 {
		t.Errorf("Unexpectedly got back %d > 0 trace exporters", len(tes))
	}
	if len(mes) != 0 {
		t.Errorf("Unexpectedly got back %d > 0 metrics exporters", len(mes))
	}
	for _, doneFn := range doneFns {
		doneFn()
	}
}

func TestPrometheusExporter_endToEnd(t *testing.T) {
	config := []byte(`
prometheus:
    namespace: "test"
    const_labels: {
        "foo": "bar",
        "code": "one"
    }
    address: ":7777"
`)

	v, _ := viperutils.ViperFromYAMLBytes([]byte(config))
	_, mes, doneFns, err := PrometheusExportersFromViper(v)
	defer func() {
		for _, doneFn := range doneFns {
			doneFn()
		}
	}()

	if err != nil {
		t.Fatalf("Unexpected parse error: %v", err)
	}
	if len(mes) == 0 {
		t.Fatal("Unexpectedly got back 0 metrics exporters")
	}

	var metric1 = &metricspb.Metric{
		Descriptor_: &metricspb.Metric_MetricDescriptor{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        "this/one/there(where)",
				Description: "Extra ones",
				Unit:        "1",
				LabelKeys: []*metricspb.LabelKey{
					{Key: "os", Description: "Operating system"},
					{Key: "arch", Description: "Architecture"},
				},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				StartTimestamp: &timestamp.Timestamp{
					Seconds: 1543160298,
					Nanos:   100000090,
				},
				LabelValues: []*metricspb.LabelValue{
					{Value: "windows"},
					{Value: "x86"},
				},
				Points: []*metricspb.Point{
					{
						Timestamp: &timestamp.Timestamp{
							Seconds: 1543160298,
							Nanos:   100000997,
						},
						Value: &metricspb.Point_Int64Value{
							Int64Value: 99,
						},
					},
				},
			},
		},
	}
	me := mes[0]
	me.ProcessMetricsData(context.Background(), data.MetricsData{Metrics: []*metricspb.Metric{metric1}})

	res, err := http.Get("http://localhost:7777/metrics")
	if err != nil {
		t.Fatalf("Failed to perform a scrape: %v", err)
	}
	if g, w := res.StatusCode, 200; g != w {
		t.Errorf("Mismatched HTTP response status code: Got: %d Want: %d", g, w)
	}
	blob, _ := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	want := `# HELP test_this_one_there_where_ Extra ones
# TYPE test_this_one_there_where_ counter
test_this_one_there_where_{arch="x86",code="one",foo="bar",os="windows"} 99
`
	if got := string(blob); got != want {
		t.Errorf("Response mismatch\nGot:\n%s\n\nWant:\n%s", got, want)
	}
}
