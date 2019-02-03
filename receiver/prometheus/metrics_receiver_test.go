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

package prometheus_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	promreceiver "github.com/census-instrumentation/opencensus-service/receiver/prometheus"
	"github.com/census-instrumentation/opencensus-service/receiver/testhelper"
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

	config := new(promreceiver.Configuration)
	if err := yaml.Unmarshal([]byte(yamlConfig), config); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	precv, err := promreceiver.New(config)
	if err != nil {
		t.Fatalf("Failed to create promreceiver: %v", err)
	}

	cms := new(testhelper.ConcurrentMetricsSink)
	if err := precv.StartMetricsReception(context.Background(), cms); err != nil {
		t.Fatalf("Failed to invoke StartMetricsReception: %v", err)
	}
	defer precv.StopMetricsReception(context.Background())

	keyMethod, _ := tag.NewKey("method")
	lm := stats.Float64("call_latency", "The latency in milliseconds per call", "ms")
	ldv := &view.View{
		Name:        "e2e/call_latency",
		Description: "The latency in milliseconds per call",
		Aggregation: view.Distribution(0, 100, 200, 500, 1000, 5000),
		Measure:     lm,
		TagKeys:     []tag.Key{keyMethod},
	}
	lcv := &view.View{
		Name:        "e2e/calls",
		Description: "The number of calls",
		Aggregation: view.Count(),
		Measure:     lm,
		TagKeys:     []tag.Key{keyMethod},
	}

	now := time.Now()
	nowPlus15ms := now.Add(15 * time.Millisecond)
	vdl := []*view.Data{
		{
			View:  ldv,
			Start: now, End: nowPlus15ms,
			Rows: []*view.Row{
				{
					Tags: []tag.Tag{{Key: keyMethod, Value: "a.b.c.run"}},
					Data: &view.DistributionData{
						Count:           1,
						Min:             180.9,
						Max:             180.9,
						SumOfSquaredDev: 0,
						CountPerBucket:  []int64{0, 1, 0, 0, 0, 0},
					},
				},
			},
		},
		{
			View:  lcv,
			Start: now, End: nowPlus15ms,
			Rows: []*view.Row{
				{
					Tags: []tag.Tag{{Key: keyMethod, Value: "a.b.c.run"}},
					Data: &view.CountData{Value: 1},
				},
			},
		},
	}

	// Perform the export as a program that exports to Prometheus would.
	for _, vd := range vdl {
		pe.ExportView(vd)
	}

	// Wait for the first scrape.
	<-time.After(500 * time.Millisecond)
	<-scrapeTrackCh
	precv.Flush()
	<-scrapeTrackCh
	// Pause for the next scrape
	precv.Flush()

	close(shutdownCh)

	got := cms.AllMetrics()

	// Unfortunately we can't control the time that Prometheus produces from scraping,
	// hence for equality, we manually have to retrieve the times recorded by Prometheus,
	// but indexed by each unique MetricDescriptor.Name().
	retrievedTimestamps := indexTimestampsByMetricDescriptorName(got)

	// Now compare the received metrics data with what we expect.
	want1 := []*agentmetricspb.ExportMetricsServiceRequest{
		{
			Node: &commonpb.Node{
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
			},
			Metrics: []*metricspb.Metric{
				{
					Descriptor_: &metricspb.Metric_MetricDescriptor{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:        "e2ereceiver_e2e_calls",
							Description: "The number of calls",
							Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
							LabelKeys: []*metricspb.LabelKey{
								{Key: "method"},
							},
						},
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: retrievedTimestamps["e2ereceiver_e2e_calls"],
							LabelValues: []*metricspb.LabelValue{
								{Value: "a.b.c.run"},
							},
							Points: []*metricspb.Point{
								{
									Timestamp: retrievedTimestamps["e2ereceiver_e2e_calls"],
									Value: &metricspb.Point_Int64Value{
										Int64Value: 1,
									},
								},
							},
						},
					},
				},
				{
					Descriptor_: &metricspb.Metric_MetricDescriptor{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:        "e2ereceiver_e2e_call_latency",
							Description: "The latency in milliseconds per call",
							Type:        metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{
								{Key: "method"},
							},
						},
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: retrievedTimestamps["e2ereceiver_e2e_call_latency"],
							LabelValues: []*metricspb.LabelValue{
								{Value: "a.b.c.run"},
							},
							Points: []*metricspb.Point{
								{
									Timestamp: retrievedTimestamps["e2ereceiver_e2e_call_latency"],
									Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											Count:                 1,
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
				},
			},
		},
	}

	want2 := []*agentmetricspb.ExportMetricsServiceRequest{
		{
			Node: &commonpb.Node{
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
			},
			Metrics: []*metricspb.Metric{
				{
					Descriptor_: &metricspb.Metric_MetricDescriptor{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:        "e2ereceiver_e2e_calls",
							Description: "The number of calls",
							Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
							LabelKeys: []*metricspb.LabelKey{
								{Key: "method"},
							},
						},
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: retrievedTimestamps["e2ereceiver_e2e_calls"],
							LabelValues: []*metricspb.LabelValue{
								{Value: "a.b.c.run"},
							},
							Points: []*metricspb.Point{
								{
									Timestamp: retrievedTimestamps["e2ereceiver_e2e_calls"],
									Value: &metricspb.Point_Int64Value{
										Int64Value: 1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Node: &commonpb.Node{
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
			},
			Metrics: []*metricspb.Metric{
				{
					Descriptor_: &metricspb.Metric_MetricDescriptor{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:        "e2ereceiver_e2e_call_latency",
							Description: "The latency in milliseconds per call",
							Type:        metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{
								{Key: "method"},
							},
						},
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: retrievedTimestamps["e2ereceiver_e2e_call_latency"],
							LabelValues: []*metricspb.LabelValue{
								{Value: "a.b.c.run"},
							},
							Points: []*metricspb.Point{
								{
									Timestamp: retrievedTimestamps["e2ereceiver_e2e_call_latency"],
									Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											Count:                 1,
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
				},
			},
		},
	}

	// Firstly sort them so that comparisons return stable results.

	byMetricsSorter(t, got)
	byMetricsSorter(t, want1)
	byMetricsSorter(t, want2)

	// Since these tests rely on underdeterministic behavior and timing that's imprecise.
	// The best that we can do is provide any of variants of what we want.
	wantPermutations := [][]*agentmetricspb.ExportMetricsServiceRequest{
		want1, want2,
	}

	for _, want := range wantPermutations {
		if !reflect.DeepEqual(got, want) {
			gj, wj := string(testhelper.ToJSON(got)), string(testhelper.ToJSON(want))
			if gj == wj {
				return
			}
		}
	}

	// Otherwise no variant of the wanted data matched, hence error out.

	gj := testhelper.ToJSON(got)
	for _, want := range wantPermutations {
		wj := testhelper.ToJSON(want)
		t.Errorf("Failed to match either:\nGot:\n%s\n\nWant:\n%s\n\n", gj, wj)
	}

}

func byMetricsSorter(t *testing.T, ereqs []*agentmetricspb.ExportMetricsServiceRequest) {
	for i, ereq := range ereqs {
		eMetrics := ereq.Metrics
		sort.Slice(eMetrics, func(i, j int) bool {
			emi, emj := eMetrics[i], eMetrics[j]
			return emi.GetMetricDescriptor().GetName() < emj.GetMetricDescriptor().GetName()
		})
		ereq.Metrics = eMetrics
		ereqs[i] = ereq
	}

	// Then sort by requests.
	sort.Slice(ereqs, func(i, j int) bool {
		eir, ejr := ereqs[i], ereqs[j]
		return eir.String() < ejr.String()
	})
}

func indexTimestampsByMetricDescriptorName(ereqs []*agentmetricspb.ExportMetricsServiceRequest) map[string]*timestamp.Timestamp {
	index := make(map[string]*timestamp.Timestamp)
	for _, ereq := range ereqs {
		if ereq == nil {
			continue
		}
		for _, eimetric := range ereq.Metrics {
			for _, eiTimeSeries := range eimetric.Timeseries {
				if ts := eiTimeSeries.GetStartTimestamp(); ts != nil {
					index[eimetric.GetMetricDescriptor().GetName()] = ts
					break
				}
			}
		}
	}
	return index
}
