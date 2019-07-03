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
	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/spf13/viper"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-service/internal/config/viperutils"
	"github.com/open-telemetry/opentelemetry-service/receiver/receivertest"
)

var logger, _ = zap.NewDevelopment()

type scrapeCounter struct {
	scrapeTrackCh chan bool
	shutdownCh    <-chan bool
	pe            http.Handler
}

func (sc *scrapeCounter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	select {
	case <-sc.shutdownCh:
		http.Error(rw, "shutting down", http.StatusGone)

	default:
		sc.scrapeTrackCh <- true
		sc.pe.ServeHTTP(rw, req)
	}
}

func TestNew(t *testing.T) {
	v := viper.New()

	_, err := New(logger, v, nil)
	if err != errNilScrapeConfig {
		t.Fatalf("Expected errNilScrapeConfig but did not get it.")
	}

	v.Set("config", nil)
	_, err = New(logger, v, nil)
	if err != errNilScrapeConfig {
		t.Fatalf("Expected errNilScrapeConfig but did not get it.")
	}

	v.Set("config.blah", "some_value")
	_, err = New(logger, v, nil)
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
	precv, err := New(logger, v, cms)
	if err != nil {
		t.Fatalf("Failed to create promreceiver: %v", err)
	}

	mh := receivertest.NewMockHost()
	if err := precv.StartMetricsReception(mh); err != nil {
		t.Fatalf("Failed to invoke StartMetricsReception: %v", err)
	}
	defer precv.StopMetricsReception()

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

	got := cms.AllMetrics()

	// Unfortunately we can't control the time that Prometheus produces from scraping,
	// hence for equality, we manually have to retrieve the times recorded by Prometheus,
	// but indexed by each unique MetricDescriptor.Name().
	retrievedTimestamps := indexTimestampsByMetricDescriptorName(got)

	// Now compare the received metrics data with what we expect.
	want1 := []consumerdata.MetricsData{
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
						Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
						LabelKeys: []*metricspb.LabelKey{
							{Key: "method"},
						},
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: retrievedTimestamps["e2ereceiver_e2e_calls"],
							LabelValues: []*metricspb.LabelValue{
								{Value: "a.b.c.run", HasValue: true},
							},
							Points: []*metricspb.Point{
								{
									Timestamp: retrievedTimestamps["e2ereceiver_e2e_calls"],
									Value: &metricspb.Point_DoubleValue{
										DoubleValue: 1,
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
								{Value: "a.b.c.run", HasValue: true},
							},
							Points: []*metricspb.Point{
								{
									Timestamp: retrievedTimestamps["e2ereceiver_e2e_call_latency"],
									Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											Count: 1,
											Sum:   180,
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{0, 100, 200, 500, 1000, 5000},
													},
												},
											},
											// size(bounds) + 1 (= N) buckets. The boundaries for bucket
											Buckets: []*metricspb.DistributionValue_Bucket{
												{},
												{Count: 1},
												{},
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

	want2 := []consumerdata.MetricsData{
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
						Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
						LabelKeys: []*metricspb.LabelKey{
							{Key: "method"},
						},
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: retrievedTimestamps["e2ereceiver_e2e_calls"],
							LabelValues: []*metricspb.LabelValue{
								{Value: "a.b.c.run", HasValue: true},
							},
							Points: []*metricspb.Point{
								{
									Timestamp: retrievedTimestamps["e2ereceiver_e2e_calls"],
									Value: &metricspb.Point_DoubleValue{
										DoubleValue: 1,
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
								{Value: "a.b.c.run", HasValue: true},
							},
							Points: []*metricspb.Point{
								{
									Timestamp: retrievedTimestamps["e2ereceiver_e2e_call_latency"],
									Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											Count: 1,
											Sum:   180,
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{0, 100, 200, 500, 1000, 5000},
													},
												},
											},
											// size(bounds) + 1 (= N) buckets. The boundaries for bucket
											Buckets: []*metricspb.DistributionValue_Bucket{
												{},
												{Count: 1},
												{},
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

	if reflect.DeepEqual(got, want1) {
		fmt.Println("done")
		return
	}

	// Since these tests rely on underdeterministic behavior and timing that's imprecise.
	// The best that we can do is provide any of variants of what we want.
	wantPermutations := [][]consumerdata.MetricsData{
		want1, want2,
	}

	for _, want := range wantPermutations {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("different metric got:\n%v\nwant:\n%v\n", string(exportertest.ToJSON(got)),
				string(exportertest.ToJSON(want)))
		}
	}
}

func byMetricsSorter(t *testing.T, mds []consumerdata.MetricsData) {
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

func indexTimestampsByMetricDescriptorName(mds []consumerdata.MetricsData) map[string]*timestamp.Timestamp {
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

type fakeProducer struct {
	metrics []*metricdata.Metric
}

func (producer *fakeProducer) Read() []*metricdata.Metric {
	return producer.metrics
}
