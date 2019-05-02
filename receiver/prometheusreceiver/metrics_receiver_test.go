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

package prometheusreceiver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/spf13/viper"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/census-instrumentation/opencensus-service/exporter/exportertest"
	"github.com/census-instrumentation/opencensus-service/internal/config/viperutils"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type scrapeCounter struct {
	scrapeTrackCh chan bool
	shutdownCh    <-chan bool
	pe            http.Handler
}

func (sc *scrapeCounter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	select {
	case <-sc.shutdownCh:
		http.Error(rw, "shuting down", http.StatusGone)

	default:
		sc.scrapeTrackCh <- true
		sc.pe.ServeHTTP(rw, req)
	}
}

func TestNew(t *testing.T) {
	v := viper.New()

	_, err := New(v, nil)
	if err != errNilScrapeConfig {
		t.Fatalf("Expected errNilScrapeConfig but did not get it.")
	}

	v.Set("config", nil)
	_, err = New(v, nil)
	if err != errNilScrapeConfig {
		t.Fatalf("Expected errNilScrapeConfig but did not get it.")
	}

	v.Set("config.blah", "some_value")
	_, err = New(v, nil)
	if err != errNilScrapeConfig {
		t.Fatalf("Expected errNilScrapeConfig but did not get it.")
	}
}

func TestEndToEnd(t *testing.T) {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "e2ereceiver",
	})
	if err != nil {
		t.Fatalf("Failed to create Prometheus view.Exporter: %v", err)
	}

	scrapeTrackCh := make(chan bool, 1)
	shutdownCh := make(chan bool)
	sLatch := &scrapeCounter{
		pe:            pe,
		scrapeTrackCh: scrapeTrackCh,
		shutdownCh:    shutdownCh,
	}

	cst := httptest.NewServer(sLatch)
	defer cst.Close()

	scrapePeriod := 35 * time.Millisecond

	cstURL, _ := url.Parse(cst.URL)
	yamlConfig := fmt.Sprintf(`
config:
  scrape_configs:
    - job_name: 'demo'

      scrape_interval: %s

      static_configs:
        - targets: ['%s']

buffer_period: 500ms
buffer_count: 2
`, scrapePeriod, cstURL.Host)

	host, port, _ := net.SplitHostPort(cstURL.Host)

	v := viper.New()
	if err = viperutils.LoadYAMLBytes(v, []byte(yamlConfig)); err != nil {
		t.Fatalf("Failed to load yaml config into viper")
	}

	cms := new(exportertest.SinkMetricsExporter)
	precv, err := New(v, cms)
	if err != nil {
		t.Fatalf("Failed to create promreceiver: %v", err)
	}

	if err := precv.StartMetricsReception(context.Background(), nil); err != nil {
		t.Fatalf("Failed to invoke StartMetricsReception: %v", err)
	}
	defer precv.StopMetricsReception(context.Background())

	keyMethod := metricdata.LabelKey{Key: "method"}
	ldm := metricdata.Descriptor{
		Name:        "e2e/call_latency",
		Description: "The latency in milliseconds per call",
		Unit:        "ms",
		Type:        metricdata.TypeCumulativeDistribution,
		LabelKeys:   []metricdata.LabelKey{keyMethod},
	}
	lcm := metricdata.Descriptor{
		Name:        "e2e/calls",
		Description: "The number of calls",
		Unit:        "1",
		Type:        metricdata.TypeCumulativeInt64,
		LabelKeys:   []metricdata.LabelKey{keyMethod},
	}

	valueMethod := metricdata.NewLabelValue("a.b.c.run")
	distribution := &metricdata.Distribution{
		Count:                 1,
		Sum:                   180,
		SumOfSquaredDeviation: 0,
		BucketOptions:         &metricdata.BucketOptions{Bounds: []float64{0, 100, 200, 500, 1000, 5000}},
		Buckets:               []metricdata.Bucket{{Count: 0}, {Count: 1}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}},
	}

	now := time.Now()
	nowPlus15ms := now.Add(15 * time.Millisecond)
	metricsToProduce := []*metricdata.Metric{
		{
			Descriptor: ldm,
			TimeSeries: []*metricdata.TimeSeries{
				{
					LabelValues: []metricdata.LabelValue{valueMethod},
					StartTime:   now,
					Points: []metricdata.Point{
						metricdata.NewDistributionPoint(nowPlus15ms, distribution),
					},
				},
			},
		},
		{
			Descriptor: lcm,
			TimeSeries: []*metricdata.TimeSeries{
				{
					LabelValues: []metricdata.LabelValue{valueMethod},
					StartTime:   now,
					Points: []metricdata.Point{
						metricdata.NewInt64Point(nowPlus15ms, 1),
					},
				},
			},
		},
	}

	// Perform the export as a program that exports to Prometheus would.
	producer := &fakeProducer{metrics: metricsToProduce}
	metricproducer.GlobalManager().AddProducer(producer)

	// Wait for the first scrape.
	<-time.After(500 * time.Millisecond)
	<-scrapeTrackCh
	precv.Flush()
	<-scrapeTrackCh
	// Pause for the next scrape
	precv.Flush()

	close(shutdownCh)
	gotMDs := cms.AllMetrics()
	if len(gotMDs) == 0 {
		t.Errorf("Want at least one Metric. Got zero.")
	}

	// Now compare the received metrics data with what we expect.
	wantNode := &commonpb.Node{
		Identifier: &commonpb.ProcessIdentifier{
			HostName: host,
		},
		ServiceInfo: &commonpb.ServiceInfo{
			Name: "demo",
		},
		Attributes: map[string]string{
			"scheme": "http",
			"port":   port,
		},
	}

	wantMetricPb1 := &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name:        "e2ereceiver_e2e_calls",
			Description: "The number of calls",
			Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
			LabelKeys: []*metricspb.LabelKey{
				{Key: "method"},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				LabelValues: []*metricspb.LabelValue{
					{Value: "a.b.c.run"},
				},
				Points: []*metricspb.Point{
					{
						Value: &metricspb.Point_Int64Value{
							Int64Value: 1,
						},
					},
				},
			},
		},
	}

	wantMetricPb2 := &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name:        "e2ereceiver_e2e_call_latency",
			Description: "The latency in milliseconds per call",
			Type:        metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
			LabelKeys: []*metricspb.LabelKey{
				{Key: "method"},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				LabelValues: []*metricspb.LabelValue{
					{Value: "a.b.c.run"},
				},
				Points: []*metricspb.Point{
					{
						Value: &metricspb.Point_DistributionValue{
							DistributionValue: &metricspb.DistributionValue{
								Count:                 1,
								Sum:                   180,
								SumOfSquaredDeviation: 1,
								BucketOptions: &metricspb.DistributionValue_BucketOptions{
									Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
										Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
											Bounds: []float64{0, 100, 200, 500, 1000, 5000},
										},
									},
								},
								Buckets: []*metricspb.DistributionValue_Bucket{
									{},
									{Count: 1},
									{},
									{},
									{},
									{},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, md := range gotMDs {
		node := md.Node
		if diff := cmpNodePb(node, wantNode); diff != "" {
			t.Errorf("Mismatch Node\n-Got +Want:\n%s", diff)
		}
		metricPbs := md.Metrics
		if len(metricPbs) != 1 {
			t.Errorf("Want 1 metric, got %d", len(metricPbs))
		}
		metricPb := metricPbs[0]
		diff1 := cmpMetricPb(metricPb, wantMetricPb1)
		diff2 := cmpMetricPb(metricPb, wantMetricPb2)
		if diff1 != "" && diff2 != "" {
			t.Errorf("Metric doesn't match with either of wanted metrics\n-Got +Want:\n%s\n-Got +Want:\n%s", diff1, diff2)
		}
	}
}

type fakeProducer struct {
	metrics []*metricdata.Metric
}

func (producer *fakeProducer) Read() []*metricdata.Metric {
	return producer.metrics
}

func cmpNodePb(got, want *commonpb.Node) string {
	// Ignore all "XXX_sizecache" fields.
	return cmp.Diff(
		got,
		want,
		cmpopts.IgnoreFields(commonpb.Node{}, "XXX_sizecache"),
		cmpopts.IgnoreFields(commonpb.ProcessIdentifier{}, "XXX_sizecache"),
		cmpopts.IgnoreFields(commonpb.ServiceInfo{}, "XXX_sizecache"))
}

func cmpMetricPb(got, want *metricspb.Metric) string {
	// Start and end time are non-deteministic. Ignore them when do the comparison.
	return cmp.Diff(
		got,
		want,
		cmpopts.IgnoreTypes(&timestamp.Timestamp{}),
		cmpopts.IgnoreFields(metricspb.MetricDescriptor{}, "XXX_sizecache"),
		cmpopts.IgnoreFields(metricspb.LabelKey{}, "XXX_sizecache"))
}
