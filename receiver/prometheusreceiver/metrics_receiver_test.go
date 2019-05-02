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
	"reflect"
	"sort"
	"testing"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/spf13/viper"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/census-instrumentation/opencensus-service/data"
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
	t.Skip("Fix this test by switching to export metrics instead of viewdata")
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
	want1 := []data.MetricsData{
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

	want2 := []data.MetricsData{
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
	wantPermutations := [][]data.MetricsData{
		want1, want2,
	}

	for _, want := range wantPermutations {
		if !reflect.DeepEqual(got, want) {
			gj, wj := string(exportertest.ToJSON(got)), string(exportertest.ToJSON(want))
			if gj == wj {
				return
			}
		}
	}

	// Otherwise no variant of the wanted data matched, hence error out.

	gj := exportertest.ToJSON(got)
	for _, want := range wantPermutations {
		wj := exportertest.ToJSON(want)
		t.Errorf("Failed to match either:\nGot:\n%s\n\nWant:\n%s\n\n", gj, wj)
	}

}

func byMetricsSorter(t *testing.T, mds []data.MetricsData) {
	for i, md := range mds {
		eMetrics := md.Metrics
		sort.Slice(eMetrics, func(i, j int) bool {
			mdi, mdj := eMetrics[i], eMetrics[j]
			return mdi.GetMetricDescriptor().GetName() < mdj.GetMetricDescriptor().GetName()
		})
		md.Metrics = eMetrics
		mds[i] = md
	}

	// Then sort by requests.
	sort.Slice(mds, func(i, j int) bool {
		mdi, mdj := mds[i], mds[j]
		return string(exportertest.ToJSON(mdi)) < string(exportertest.ToJSON(mdj))
	})
}

func indexTimestampsByMetricDescriptorName(mds []data.MetricsData) map[string]*timestamp.Timestamp {
	index := make(map[string]*timestamp.Timestamp)
	for _, md := range mds {
		for _, eimetric := range md.Metrics {
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
