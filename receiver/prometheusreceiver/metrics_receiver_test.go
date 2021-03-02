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

package prometheusreceiver

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	promcfg "github.com/prometheus/prometheus/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/translator/internaldata"
)

var logger = zap.NewNop()

type mockPrometheusResponse struct {
	code int
	data string
}

type mockPrometheus struct {
	endpoints   map[string][]mockPrometheusResponse
	accessIndex map[string]*int32
	wg          *sync.WaitGroup
	srv         *httptest.Server
}

func newMockPrometheus(endpoints map[string][]mockPrometheusResponse) *mockPrometheus {
	accessIndex := make(map[string]*int32)
	wg := &sync.WaitGroup{}
	wg.Add(len(endpoints))
	for k := range endpoints {
		v := int32(0)
		accessIndex[k] = &v
	}
	mp := &mockPrometheus{
		wg:          wg,
		accessIndex: accessIndex,
		endpoints:   endpoints,
	}
	srv := httptest.NewServer(mp)
	mp.srv = srv
	return mp
}

func (mp *mockPrometheus) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	iptr, ok := mp.accessIndex[req.URL.Path]
	if !ok {
		rw.WriteHeader(404)
		return
	}
	index := int(*iptr)
	atomic.AddInt32(iptr, 1)
	pages := mp.endpoints[req.URL.Path]
	if index >= len(pages) {
		if index == len(pages) {
			mp.wg.Done()
		}
		rw.WriteHeader(404)
		return
	}
	rw.WriteHeader(pages[index].code)
	_, _ = rw.Write([]byte(pages[index].data))
}

func (mp *mockPrometheus) Close() {
	mp.srv.Close()
}

// -------------------------
// EndToEnd Test and related
// -------------------------

var srvPlaceHolder = "__SERVER_ADDRESS__"

type testData struct {
	name         string
	pages        []mockPrometheusResponse
	node         *commonpb.Node
	resource     *resourcepb.Resource
	validateFunc func(t *testing.T, td *testData, result []internaldata.MetricsData)
}

// setupMockPrometheus to create a mocked prometheus based on targets, returning the server and a prometheus exporting
// config
func setupMockPrometheus(tds ...*testData) (*mockPrometheus, *promcfg.Config, error) {
	jobs := make([]map[string]interface{}, 0, len(tds))
	endpoints := make(map[string][]mockPrometheusResponse)
	for _, t := range tds {
		metricPath := fmt.Sprintf("/%s/metrics", t.name)
		endpoints[metricPath] = t.pages
		job := make(map[string]interface{})
		job["job_name"] = t.name
		job["metrics_path"] = metricPath
		job["scrape_interval"] = "1s"
		job["static_configs"] = []map[string]interface{}{{"targets": []string{srvPlaceHolder}}}
		jobs = append(jobs, job)
	}

	if len(jobs) != len(tds) {
		log.Fatal("len(jobs) != len(targets), make sure job names are unique")
	}
	config := make(map[string]interface{})
	config["scrape_configs"] = jobs

	mp := newMockPrometheus(endpoints)
	cfg, err := yaml.Marshal(&config)
	if err != nil {
		return mp, nil, err
	}
	u, _ := url.Parse(mp.srv.URL)
	host, port, _ := net.SplitHostPort(u.Host)

	// update node value (will use for validation)
	for _, t := range tds {
		t.node = &commonpb.Node{
			Identifier: &commonpb.ProcessIdentifier{
				HostName: host,
			},
			ServiceInfo: &commonpb.ServiceInfo{
				Name: t.name,
			},
		}
		t.resource = &resourcepb.Resource{
			Labels: map[string]string{
				"scheme": "http",
				"port":   port,
			},
		}
	}

	cfgStr := strings.ReplaceAll(string(cfg), srvPlaceHolder, u.Host)
	pCfg, err := promcfg.Load(cfgStr)
	return mp, pCfg, err
}

func verifyNumScrapeResults(t *testing.T, td *testData, mds []internaldata.MetricsData) {
	want := 0
	for _, p := range td.pages {
		if p.code == 200 {
			want++
		}
	}
	if l := len(mds); l != want {
		t.Errorf("want %d, but got %d\n", want, l)
	}
}

func doCompare(name string, t *testing.T, want, got interface{}) {
	t.Run(name, func(t *testing.T) {
		assert.EqualValues(t, want, got)
	})
}

// Test data and validation functions for EndToEnd test
// Make sure every page has a gauge, we are relying on it to figure out the starttime if needed

// target1 has one gague, two counts of a same family, one histogram and one summary. We are expecting the first
// successful scrape will produce a metric with only the gauge metric, and the 2nd successful scrape will produce all
// the metrics, with the cumulative types using the firstpage's timestamp as starttime, and values are deltas with the
// one from the first page (summary quantiles will be as it is)
var target1Page1 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 19

# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 100
http_requests_total{method="post",code="400"} 5

# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 1000
http_request_duration_seconds_bucket{le="0.5"} 1500
http_request_duration_seconds_bucket{le="1"} 2000
http_request_duration_seconds_bucket{le="+Inf"} 2500
http_request_duration_seconds_sum 5000
http_request_duration_seconds_count 2500

# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 1
rpc_duration_seconds{quantile="0.9"} 5
rpc_duration_seconds{quantile="0.99"} 8
rpc_duration_seconds_sum 5000
rpc_duration_seconds_count 1000
`

var target1Page2 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 18

# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 199
http_requests_total{method="post",code="400"} 12

# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 1100
http_request_duration_seconds_bucket{le="0.5"} 1600
http_request_duration_seconds_bucket{le="1"} 2100
http_request_duration_seconds_bucket{le="+Inf"} 2600
http_request_duration_seconds_sum 5050
http_request_duration_seconds_count 2600

# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 1
rpc_duration_seconds{quantile="0.9"} 6
rpc_duration_seconds{quantile="0.99"} 8
rpc_duration_seconds_sum 5002
rpc_duration_seconds_count 1001
`

func verifyTarget1(t *testing.T, td *testData, mds []internaldata.MetricsData) {
	verifyNumScrapeResults(t, td, mds)
	m1 := mds[0]
	// m1 shall only have a gauge
	if l := len(m1.Metrics); l != 1 {
		t.Errorf("want 1, but got %v\n", l)
	}

	// only gauge value is returned from the first scrape
	wantG1 := &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name:        "go_threads",
			Description: "Number of OS threads created",
			Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				Points: []*metricspb.Point{
					{Value: &metricspb.Point_DoubleValue{DoubleValue: 19.0}},
				},
			},
		},
	}
	gotG1 := m1.Metrics[0]
	// relying on the timestamps from gagues as startTimestamps
	ts1 := gotG1.Timeseries[0].Points[0].Timestamp
	// set this timestamp to wantG1
	wantG1.Timeseries[0].Points[0].Timestamp = ts1
	doCompare("scrape1", t, wantG1, gotG1)

	// verify the 2nd metricData
	m2 := mds[1]
	ts2 := m2.Metrics[0].Timeseries[0].Points[0].Timestamp

	want2 := &internaldata.MetricsData{
		Node:     td.node,
		Resource: td.resource,
		Metrics: []*metricspb.Metric{
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "go_threads",
					Description: "Number of OS threads created",
					Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE},
				Timeseries: []*metricspb.TimeSeries{
					{
						Points: []*metricspb.Point{
							{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 18.0}},
						},
					},
				},
			},
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "http_requests_total",
					Description: "The total number of HTTP requests.",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
					LabelKeys:   []*metricspb.LabelKey{{Key: "code"}, {Key: "method"}},
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: ts1,
						LabelValues: []*metricspb.LabelValue{
							{Value: "200", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 99.0}},
						},
					},
					{
						StartTimestamp: ts1,
						LabelValues: []*metricspb.LabelValue{
							{Value: "400", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 7.0}},
						},
					},
				},
			},
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "http_request_duration_seconds",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
					Description: "A histogram of the request duration.",
					Unit:        "s",
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: ts1,
						Points: []*metricspb.Point{
							{
								Timestamp: ts2,
								Value: &metricspb.Point_DistributionValue{
									DistributionValue: &metricspb.DistributionValue{
										BucketOptions: &metricspb.DistributionValue_BucketOptions{
											Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
												Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
													Bounds: []float64{0.05, 0.5, 1},
												},
											},
										},
										Count: 100,
										Sum:   50.0,
										Buckets: []*metricspb.DistributionValue_Bucket{
											{Count: 100},
											{Count: 0},
											{Count: 0},
											{Count: 0},
										},
									}},
							},
						},
					},
				},
			},
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "rpc_duration_seconds",
					Type:        metricspb.MetricDescriptor_SUMMARY,
					Description: "A summary of the RPC duration in seconds.",
					Unit:        "s",
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: ts1,
						Points: []*metricspb.Point{
							{
								Timestamp: ts2,
								Value: &metricspb.Point_SummaryValue{
									SummaryValue: &metricspb.SummaryValue{
										Sum:   &wrappers.DoubleValue{Value: 2},
										Count: &wrappers.Int64Value{Value: 1},
										Snapshot: &metricspb.SummaryValue_Snapshot{
											PercentileValues: []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{
												{
													Percentile: 1,
													Value:      1,
												},
												{
													Percentile: 90,
													Value:      6,
												},
												{
													Percentile: 99,
													Value:      8,
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
		},
	}

	doCompare("scrape2", t, want2, &m2)
}

// target2 is going to have 5 pages, and there's a newly appeared item from the 2nd page. we are expecting the new
// metric will appears on the 3rd scrape, and it uses the timestamp from the 2nd page as starttime, and value is delta
// from the 2nd page.
// with the 4th page, we are simulating a reset (values smaller than previous), cumulative types shall be skipped at
// this run. with the 5th page, cumulative types will be delta with the 4th one
var target2Page1 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 18

# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 10
http_requests_total{method="post",code="400"} 50
`

var target2Page2 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 16

# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 50
http_requests_total{method="post",code="400"} 60
http_requests_total{method="post",code="500"} 3
`

var target2Page3 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 16

# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 50
http_requests_total{method="post",code="400"} 60
http_requests_total{method="post",code="500"} 5
`

var target2Page4 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 16

# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 49
http_requests_total{method="post",code="400"} 59
http_requests_total{method="post",code="500"} 3
`

var target2Page5 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 16

# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 50
http_requests_total{method="post",code="400"} 59
http_requests_total{method="post",code="500"} 5
`

func verifyTarget2(t *testing.T, td *testData, mds []internaldata.MetricsData) {
	verifyNumScrapeResults(t, td, mds)
	m1 := mds[0]
	// m1 shall only have a gauge
	if l := len(m1.Metrics); l != 1 {
		t.Errorf("want 1, but got %v\n", l)
	}

	// only gauge value is returned from the first scrape
	wantG1 := &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name:        "go_threads",
			Description: "Number of OS threads created",
			Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				Points: []*metricspb.Point{
					{Value: &metricspb.Point_DoubleValue{DoubleValue: 18.0}},
				},
			},
		},
	}
	gotG1 := m1.Metrics[0]
	ts1 := gotG1.Timeseries[0].Points[0].Timestamp
	// set this timestamp to wantG1
	wantG1.Timeseries[0].Points[0].Timestamp = ts1
	doCompare("scrape1", t, wantG1, gotG1)

	// verify the 2nd metricData
	m2 := mds[1]
	ts2 := m2.Metrics[0].Timeseries[0].Points[0].Timestamp

	want2 := &internaldata.MetricsData{
		Node:     td.node,
		Resource: td.resource,
		Metrics: []*metricspb.Metric{
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "go_threads",
					Description: "Number of OS threads created",
					Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						Points: []*metricspb.Point{
							{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 16.0}},
						},
					},
				},
			},
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "http_requests_total",
					Description: "The total number of HTTP requests.",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
					LabelKeys:   []*metricspb.LabelKey{{Key: "code"}, {Key: "method"}},
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: ts1,
						LabelValues: []*metricspb.LabelValue{
							{Value: "200", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 40.0}},
						},
					},
					{
						StartTimestamp: ts1,
						LabelValues: []*metricspb.LabelValue{
							{Value: "400", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 10.0}},
						},
					},
				},
			},
		},
	}
	doCompare("scrape2", t, want2, &m2)

	// verify the 3rd metricData, with the new code=500 counter which first appeared on 2nd run
	m3 := mds[2]
	// its start timestamp shall be from the 2nd run
	ts3 := m3.Metrics[0].Timeseries[0].Points[0].Timestamp

	want3 := &internaldata.MetricsData{
		Node:     td.node,
		Resource: td.resource,
		Metrics: []*metricspb.Metric{
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "go_threads",
					Description: "Number of OS threads created",
					Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						Points: []*metricspb.Point{
							{Timestamp: ts3, Value: &metricspb.Point_DoubleValue{DoubleValue: 16.0}},
						},
					},
				},
			},
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "http_requests_total",
					Description: "The total number of HTTP requests.",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
					LabelKeys:   []*metricspb.LabelKey{{Key: "code"}, {Key: "method"}},
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: ts1,
						LabelValues: []*metricspb.LabelValue{
							{Value: "200", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts3, Value: &metricspb.Point_DoubleValue{DoubleValue: 40.0}},
						},
					},
					{
						StartTimestamp: ts1,
						LabelValues: []*metricspb.LabelValue{
							{Value: "400", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts3, Value: &metricspb.Point_DoubleValue{DoubleValue: 10.0}},
						},
					},
					{
						StartTimestamp: ts2,
						LabelValues: []*metricspb.LabelValue{
							{Value: "500", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts3, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
						},
					},
				},
			},
		},
	}
	doCompare("scrape3", t, want3, &m3)

	// verify the 4th metricData which reset happens, all cumulative types shall be absent
	m4 := mds[3]
	ts4 := m4.Metrics[0].Timeseries[0].Points[0].Timestamp

	want4 := &internaldata.MetricsData{
		Node:     td.node,
		Resource: td.resource,
		Metrics: []*metricspb.Metric{
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "go_threads",
					Description: "Number of OS threads created",
					Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						Points: []*metricspb.Point{
							{Timestamp: ts4, Value: &metricspb.Point_DoubleValue{DoubleValue: 16.0}},
						},
					},
				},
			},
		},
	}
	doCompare("scrape4", t, want4, &m4)

	// verify the 4th metricData which reset happens, all cumulative types shall be absent
	m5 := mds[4]
	// its start timestamp shall be from the 3rd run
	ts5 := m5.Metrics[0].Timeseries[0].Points[0].Timestamp

	want5 := &internaldata.MetricsData{
		Node:     td.node,
		Resource: td.resource,
		Metrics: []*metricspb.Metric{
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "go_threads",
					Description: "Number of OS threads created",
					Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						Points: []*metricspb.Point{
							{Timestamp: ts5, Value: &metricspb.Point_DoubleValue{DoubleValue: 16.0}},
						},
					},
				},
			},
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "http_requests_total",
					Description: "The total number of HTTP requests.",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
					LabelKeys:   []*metricspb.LabelKey{{Key: "code"}, {Key: "method"}},
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: ts4,
						LabelValues: []*metricspb.LabelValue{
							{Value: "200", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts5, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
						},
					},
					{
						StartTimestamp: ts4,
						LabelValues: []*metricspb.LabelValue{
							{Value: "400", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts5, Value: &metricspb.Point_DoubleValue{DoubleValue: 0.0}},
						},
					},
					{
						StartTimestamp: ts4,
						LabelValues: []*metricspb.LabelValue{
							{Value: "500", HasValue: true},
							{Value: "post", HasValue: true},
						},
						Points: []*metricspb.Point{
							{Timestamp: ts5, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
						},
					},
				},
			},
		},
	}
	doCompare("scrape5", t, want5, &m5)
}

// target3 for complicated data types, including summaries and histograms. one of the summary and histogram have only
// sum/count, for the summary it's valid, however the histogram one is not, but it shall not cause the scrape to fail
var target3Page1 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 18

# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.2"} 10000
http_request_duration_seconds_bucket{le="0.5"} 11000
http_request_duration_seconds_bucket{le="1"} 12001
http_request_duration_seconds_bucket{le="+Inf"} 13003
http_request_duration_seconds_sum 50000
http_request_duration_seconds_count 13003

# A corrupted histogram with only sum and count
# HELP corrupted_hist A corrupted_hist.
# TYPE corrupted_hist histogram
corrupted_hist_sum 100
corrupted_hist_count 10

# Finally a summary, which has a complex representation, too:
# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{foo="bar" quantile="0.01"} 31
rpc_duration_seconds{foo="bar" quantile="0.05"} 35
rpc_duration_seconds{foo="bar" quantile="0.5"} 47
rpc_duration_seconds{foo="bar" quantile="0.9"} 70
rpc_duration_seconds{foo="bar" quantile="0.99"} 76
rpc_duration_seconds_sum{foo="bar"} 8000
rpc_duration_seconds_count{foo="bar"} 900
rpc_duration_seconds_sum{foo="no_quantile"} 100
rpc_duration_seconds_count{foo="no_quantile"} 50
`

var target3Page2 = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 16

# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.2"} 11000
http_request_duration_seconds_bucket{le="0.5"} 12000
http_request_duration_seconds_bucket{le="1"} 13001
http_request_duration_seconds_bucket{le="+Inf"} 14003
http_request_duration_seconds_sum 50100
http_request_duration_seconds_count 14003

# A corrupted histogram with only sum and count
# HELP corrupted_hist A corrupted_hist.
# TYPE corrupted_hist histogram
corrupted_hist_sum 101
corrupted_hist_count 15

# Finally a summary, which has a complex representation, too:
# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{foo="bar" quantile="0.01"} 32
rpc_duration_seconds{foo="bar" quantile="0.05"} 35
rpc_duration_seconds{foo="bar" quantile="0.5"} 47
rpc_duration_seconds{foo="bar" quantile="0.9"} 70
rpc_duration_seconds{foo="bar" quantile="0.99"} 77
rpc_duration_seconds_sum{foo="bar"} 8100
rpc_duration_seconds_count{foo="bar"} 950
rpc_duration_seconds_sum{foo="no_quantile"} 101
rpc_duration_seconds_count{foo="no_quantile"} 55
`

func verifyTarget3(t *testing.T, td *testData, mds []internaldata.MetricsData) {
	verifyNumScrapeResults(t, td, mds)
	m1 := mds[0]
	// m1 shall only have a gauge
	if l := len(m1.Metrics); l != 1 {
		t.Errorf("want 1, but got %v\n", l)
	}

	// only gauge value is returned from the first scrape
	wantG1 := &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name:        "go_threads",
			Description: "Number of OS threads created",
			Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE},
		Timeseries: []*metricspb.TimeSeries{
			{
				Points: []*metricspb.Point{
					{Value: &metricspb.Point_DoubleValue{DoubleValue: 18.0}},
				},
			},
		},
	}
	gotG1 := m1.Metrics[0]
	ts1 := gotG1.Timeseries[0].Points[0].Timestamp
	// set this timestamp to wantG1
	wantG1.Timeseries[0].Points[0].Timestamp = ts1
	doCompare("scrape1", t, wantG1, gotG1)

	// verify the 2nd metricData
	m2 := mds[1]
	ts2 := m2.Metrics[0].Timeseries[0].Points[0].Timestamp

	want2 := &internaldata.MetricsData{
		Node:     td.node,
		Resource: td.resource,
		Metrics: []*metricspb.Metric{
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "go_threads",
					Description: "Number of OS threads created",
					Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						Points: []*metricspb.Point{
							{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 16.0}},
						},
					},
				},
			},
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "http_request_duration_seconds",
					Description: "A histogram of the request duration.",
					Unit:        "s",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: ts1,
						Points: []*metricspb.Point{
							{
								Timestamp: ts2,
								Value: &metricspb.Point_DistributionValue{
									DistributionValue: &metricspb.DistributionValue{
										BucketOptions: &metricspb.DistributionValue_BucketOptions{
											Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
												Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
													Bounds: []float64{0.2, 0.5, 1},
												},
											},
										},
										Count: 1000,
										Sum:   100,
										Buckets: []*metricspb.DistributionValue_Bucket{
											{Count: 1000},
											{Count: 0},
											{Count: 0},
											{Count: 0},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "rpc_duration_seconds",
					Type:        metricspb.MetricDescriptor_SUMMARY,
					LabelKeys:   []*metricspb.LabelKey{{Key: "foo"}},
					Description: "A summary of the RPC duration in seconds.",
					Unit:        "s",
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: ts1,
						LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
						Points: []*metricspb.Point{
							{
								Timestamp: ts2,
								Value: &metricspb.Point_SummaryValue{
									SummaryValue: &metricspb.SummaryValue{
										Sum:   &wrappers.DoubleValue{Value: 100},
										Count: &wrappers.Int64Value{Value: 50},
										Snapshot: &metricspb.SummaryValue_Snapshot{
											PercentileValues: []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{
												{
													Percentile: 1,
													Value:      32,
												},
												{
													Percentile: 5,
													Value:      35,
												},
												{
													Percentile: 50,
													Value:      47,
												},
												{
													Percentile: 90,
													Value:      70,
												},
												{
													Percentile: 99,
													Value:      77,
												},
											},
										},
									},
								},
							},
						},
					},
					{
						StartTimestamp: ts1,
						LabelValues:    []*metricspb.LabelValue{{Value: "no_quantile", HasValue: true}},
						Points: []*metricspb.Point{
							{
								Timestamp: ts2,
								Value: &metricspb.Point_SummaryValue{
									SummaryValue: &metricspb.SummaryValue{
										Sum:   &wrappers.DoubleValue{Value: 1},
										Count: &wrappers.Int64Value{Value: 5},
										Snapshot: &metricspb.SummaryValue_Snapshot{
											PercentileValues: nil,
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

	doCompare("scrape2", t, want2, &m2)
}

// TestEndToEnd  end to end test executor
func TestEndToEnd(t *testing.T) {
	// 1. setup input data
	targets := []*testData{
		{
			name: "target1",
			pages: []mockPrometheusResponse{
				{code: 200, data: target1Page1},
				{code: 500, data: ""},
				{code: 200, data: target1Page2},
			},
			validateFunc: verifyTarget1,
		},
		{
			name: "target2",
			pages: []mockPrometheusResponse{
				{code: 200, data: target2Page1},
				{code: 200, data: target2Page2},
				{code: 500, data: ""},
				{code: 200, data: target2Page3},
				{code: 200, data: target2Page4},
				{code: 500, data: ""},
				{code: 200, data: target2Page5},
			},
			validateFunc: verifyTarget2,
		},
		{
			name: "target3",
			pages: []mockPrometheusResponse{
				{code: 200, data: target3Page1},
				{code: 200, data: target3Page2},
			},
			validateFunc: verifyTarget3,
		},
	}

	testEndToEnd(t, targets, false)
}

var startTimeMetricPage = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 19
# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 100
http_requests_total{method="post",code="400"} 5
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 1000
http_request_duration_seconds_bucket{le="0.5"} 1500
http_request_duration_seconds_bucket{le="1"} 2000
http_request_duration_seconds_bucket{le="+Inf"} 2500
http_request_duration_seconds_sum 5000
http_request_duration_seconds_count 2500
# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 1
rpc_duration_seconds{quantile="0.9"} 5
rpc_duration_seconds{quantile="0.99"} 8
rpc_duration_seconds_sum 5000
rpc_duration_seconds_count 1000
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 400.8
`

var startTimeMetricPageStartTimestamp = &timestamppb.Timestamp{Seconds: 400, Nanos: 800000000}

const numStartTimeMetricPageTimeseries = 6

func verifyStartTimeMetricPage(t *testing.T, _ *testData, mds []internaldata.MetricsData) {
	numTimeseries := 0
	for _, cmd := range mds {
		for _, metric := range cmd.Metrics {
			timestamp := startTimeMetricPageStartTimestamp
			switch metric.GetMetricDescriptor().GetType() {
			case metricspb.MetricDescriptor_GAUGE_DOUBLE, metricspb.MetricDescriptor_GAUGE_DISTRIBUTION:
				timestamp = nil
			}
			for _, ts := range metric.GetTimeseries() {
				assert.Equal(t, timestamp, ts.GetStartTimestamp())
				numTimeseries++
			}
		}
	}
	assert.Equal(t, numStartTimeMetricPageTimeseries, numTimeseries)
}

// TestStartTimeMetric validates that timeseries have start time set to 'process_start_time_seconds'
func TestStartTimeMetric(t *testing.T) {
	targets := []*testData{
		{
			name: "target1",
			pages: []mockPrometheusResponse{
				{code: 200, data: startTimeMetricPage},
			},
			validateFunc: verifyStartTimeMetricPage,
		},
	}
	testEndToEnd(t, targets, true)
}

func testEndToEnd(t *testing.T, targets []*testData, useStartTimeMetric bool) {
	// 1. setup mock server
	mp, cfg, err := setupMockPrometheus(targets...)
	require.Nilf(t, err, "Failed to create Promtheus config: %v", err)
	defer mp.Close()

	cms := new(consumertest.MetricsSink)
	rcvr := newPrometheusReceiver(logger, &Config{PrometheusConfig: cfg, UseStartTimeMetric: useStartTimeMetric}, cms)

	require.NoError(t, rcvr.Start(context.Background(), componenttest.NewNopHost()), "Failed to invoke Start: %v", err)
	defer rcvr.Shutdown(context.Background())

	// wait for all provided data to be scraped
	mp.wg.Wait()
	metrics := cms.AllMetrics()

	// split and store results by target name
	results := make(map[string][]internaldata.MetricsData)
	for _, m := range metrics {
		ocmds := internaldata.MetricsToOC(m)
		for _, ocmd := range ocmds {
			result, ok := results[ocmd.Node.ServiceInfo.Name]
			if !ok {
				result = make([]internaldata.MetricsData, 0)
			}
			results[ocmd.Node.ServiceInfo.Name] = append(result, ocmd)
		}
	}

	lres, lep := len(results), len(mp.endpoints)
	assert.Equalf(t, lep, lres, "want %d targets, but got %v\n", lep, lres)

	// loop to validate outputs for each targets
	for _, target := range targets {
		target.validateFunc(t, target, results[target.name])
	}
}

var startTimeMetricRegexPage = `
# HELP go_threads Number of OS threads created
# TYPE go_threads gauge
go_threads 19
# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 100
http_requests_total{method="post",code="400"} 5
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 1000
http_request_duration_seconds_bucket{le="0.5"} 1500
http_request_duration_seconds_bucket{le="1"} 2000
http_request_duration_seconds_bucket{le="+Inf"} 2500
http_request_duration_seconds_sum 5000
http_request_duration_seconds_count 2500
# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 1
rpc_duration_seconds{quantile="0.9"} 5
rpc_duration_seconds{quantile="0.99"} 8
rpc_duration_seconds_sum 5000
rpc_duration_seconds_count 1000
# HELP example_process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE example_process_start_time_seconds gauge
example_process_start_time_seconds 400.8
`

// TestStartTimeMetricRegex validates that timeseries have start time regex set to 'process_start_time_seconds'
func TestStartTimeMetricRegex(t *testing.T) {
	targets := []*testData{
		{
			name: "target1",
			pages: []mockPrometheusResponse{
				{code: 200, data: startTimeMetricRegexPage},
			},
			validateFunc: verifyStartTimeMetricPage,
		},
		{
			name: "target2",
			pages: []mockPrometheusResponse{
				{code: 200, data: startTimeMetricPage},
			},
			validateFunc: verifyStartTimeMetricPage,
		},
	}
	testEndToEndRegex(t, targets, true, "^(.+_)*process_start_time_seconds$")
}

func testEndToEndRegex(t *testing.T, targets []*testData, useStartTimeMetric bool, startTimeMetricRegex string) {
	// 1. setup mock server
	mp, cfg, err := setupMockPrometheus(targets...)
	require.Nilf(t, err, "Failed to create Promtheus config: %v", err)
	defer mp.Close()

	cms := new(consumertest.MetricsSink)
	rcvr := newPrometheusReceiver(logger, &Config{PrometheusConfig: cfg, UseStartTimeMetric: useStartTimeMetric, StartTimeMetricRegex: startTimeMetricRegex}, cms)

	require.NoError(t, rcvr.Start(context.Background(), componenttest.NewNopHost()), "Failed to invoke Start: %v", err)
	defer rcvr.Shutdown(context.Background())

	// wait for all provided data to be scraped
	mp.wg.Wait()
	metrics := cms.AllMetrics()

	// split and store results by target name
	results := make(map[string][]internaldata.MetricsData)
	for _, m := range metrics {
		ocmds := internaldata.MetricsToOC(m)
		for _, ocmd := range ocmds {
			result, ok := results[ocmd.Node.ServiceInfo.Name]
			if !ok {
				result = make([]internaldata.MetricsData, 0)
			}
			results[ocmd.Node.ServiceInfo.Name] = append(result, ocmd)
		}
	}

	lres, lep := len(results), len(mp.endpoints)
	assert.Equalf(t, lep, lres, "want %d targets, but got %v\n", lep, lres)

	// loop to validate outputs for each targets
	for _, target := range targets {
		target.validateFunc(t, target, results[target.name])
	}
}
