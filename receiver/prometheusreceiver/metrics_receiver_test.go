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

	gokitlog "github.com/go-kit/kit/log"
	promcfg "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/scrape"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/translator/conventions/v1.5.0"
)

var logger = zap.NewNop()

type mockPrometheusResponse struct {
	code int
	data string
}

type mockPrometheus struct {
	mu          sync.Mutex // mu protects the fields below.
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
	mp.mu.Lock()
	defer mp.mu.Unlock()

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

var (
	srvPlaceHolder = "__SERVER_ADDRESS__"
)

type testData struct {
	name         string
	pages        []mockPrometheusResponse
	resource     *pdata.Resource
	validateFunc func(t *testing.T, td *testData, result []*pdata.MetricSlice)
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
		rs := pdata.NewResource()
		attrs := rs.Attributes()
		attrs.UpsertString(conventions.AttributeHostName, host)
		attrs.UpsertString(conventions.AttributeServiceName, t.name)
		attrs.UpsertString("job", t.name)
		attrs.UpsertString("instance", u.Host)
		attrs.UpsertString("scheme", "http")
		attrs.UpsertString("port", port)
		t.resource = &rs
	}

	cfgStr := strings.ReplaceAll(string(cfg), srvPlaceHolder, u.Host)
	pCfg, err := promcfg.Load(cfgStr, false, gokitlog.NewNopLogger())
	return mp, pCfg, err
}

const expectedScrapeMetricCount = 5

func getValidScrapes(t *testing.T, rmL []*pdata.ResourceMetrics) (out []*pdata.MetricSlice) {
	for _, rm := range rmL {
		ilmL := rm.InstrumentationLibraryMetrics()
		for i := 0; i < ilmL.Len(); i++ {
			ilm := ilmL.At(i)
			iM := ilm.Metrics()
			metricL := &iM
			// mds will include scrapes that received no metrics but have internal scrape metrics, filter those out
			if expectedScrapeMetricCount < metricL.Len() && countScrapeMetrics(metricL) == expectedScrapeMetricCount {
				assertUp(t, 1, metricL)
				out = append(out, metricL)
			} else {
				assertUp(t, 1, metricL)
			}
		}
	}
	return out
}

func countScrapeMetrics(metricL *pdata.MetricSlice) int {
	n := 0
	for i := 0; i < metricL.Len(); i++ {
		metric := metricL.At(i)
		switch metric.Name() {
		case "up", "scrape_duration_seconds", "scrape_samples_scraped", "scrape_samples_post_metric_relabeling", "scrape_series_added":
			n++
		default:
		}
	}
	return n
}

func assertUp(t *testing.T, expected float64, mL *pdata.MetricSlice) {
	for i := 0; i < mL.Len(); i++ {
		m := mL.At(i)
		if m.Name() != "up" {
			continue
		}

		assert.Equal(t, pdata.MetricDataTypeGauge, m.DataType())
		points := m.Gauge().DataPoints()
		assert.True(t, points.Len() > 0, "expecting at least one point")
		assert.Equal(t, expected, points.At(0).DoubleVal())
	}
	t.Error("No 'up' metric found")
}

func verifyNumScrapeResults(t *testing.T, td *testData, mds []*pdata.MetricSlice) {
	want := 0
	for _, p := range td.pages {
		if p.code == 200 {
			want++
		}
	}
	if l := len(mds); l != want {
		t.Fatalf("want %d, but got %d\n", want, l)
	}
}

type seriesExpectation struct {
	series []seriesComparator
	points []pointComparator
}

type pointComparator func(*testing.T, *pdata.Metric) bool
type seriesComparator func(*testing.T, *pdata.Metric) bool
type descriptorComparator func(*testing.T, *pdata.Metric) bool

type testExpectation func(*testing.T, *pdata.MetricSlice) bool

func pointCount(t *testing.T, m *pdata.Metric) int {
	switch dataType := m.DataType(); dataType {
	case pdata.MetricDataTypeSummary:
		return m.Summary().DataPoints().Len()
	case pdata.MetricDataTypeHistogram:
		return m.Histogram().DataPoints().Len()
	case pdata.MetricDataTypeGauge:
		return m.Gauge().DataPoints().Len()
	case pdata.MetricDataTypeSum:
		return m.Sum().DataPoints().Len()
	default:
		t.Fatalf("Unknown point kind: %s", dataType)
		return 0
	}
}

func assertMetricPresent(name string, descriptorExpectations []descriptorComparator, seriesExpectations []seriesExpectation) testExpectation {
	return func(t *testing.T, metricL *pdata.MetricSlice) bool {
		for i := 0; i < metricL.Len(); i++ {
			m := metricL.At(i)
			if name != m.Name() {
				continue
			}

			for _, de := range descriptorExpectations {
				if !de(t, &m) {
					return false
				}
			}

			if !assert.Equal(t, len(seriesExpectations), pointCount(t, &m)) {
				return false
			}

			// TODO: Translate me.
			/*
				for i, se := range seriesExpectations {
					for _, sc := range se.series {
						if !sc(t, m.Timeseries[i]) {
							return false
						}
					}
					for _, pc := range se.points {
						if !pc(t, m.Timeseries[i].Points[0]) {
							return false
						}
					}
				}
			*/
			return true
		}
		assert.Failf(t, "Unable to match metric expectation", name)
		return false
	}
}

func assertMetricAbsent(name string) testExpectation {
	return func(t *testing.T, metricL *pdata.MetricSlice) bool {
		for i := 0; i < metricL.Len(); i++ {
			m := metricL.At(i)
			if assert.Equal(t, name, m.Name()) {
				return false
			}
		}
		return true
	}
}

func compareStartTimestamp(ts pdata.Timestamp) seriesComparator {
	return func(t *testing.T, metric *pdata.Metric) bool {
		switch dataType := metric.DataType(); dataType {
		case pdata.MetricDataTypeGauge:
			dataPoints := metric.Gauge().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !assert.Equal(t, ts.AsTime(), point.StartTimestamp().AsTime()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeHistogram:
			dataPoints := metric.Histogram().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !assert.Equal(t, ts.AsTime(), point.StartTimestamp().AsTime()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeSummary:
			dataPoints := metric.Summary().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !assert.Equal(t, ts.AsTime(), point.StartTimestamp().AsTime()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeSum:
			dataPoints := metric.Sum().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !assert.Equal(t, ts.AsTime(), point.StartTimestamp().AsTime()) {
					return false
				}
			}
			return true

		default:
			t.Fatalf("Unhandled data type: %s", dataType)
			return false
		}
	}
}

func compareMetricType(want pdata.MetricDataType) descriptorComparator {
	return func(t *testing.T, metric *pdata.Metric) bool {
		return assert.Equal(t, want, metric.DataType())
	}
}

func ensureMatchingKeys(t *testing.T, lookup map[string]bool, sm pdata.StringMap) bool {
	if !assert.Equal(t, len(lookup), sm.Len()) {
		return false
	}
	ok := true
	sm.Range(func(key, _ string) bool {
		if _, match := lookup[key]; !match {
			ok = false
		}
		return ok
	})
	return ok
}

func ensureMatchingValues(t *testing.T, lookup map[string]bool, sm pdata.StringMap) bool {
	if !assert.Equal(t, len(lookup), sm.Len()) {
		return false
	}
	ok := true
	sm.Range(func(_, value string) bool {
		if _, match := lookup[value]; !match {
			ok = false
		}
		return ok
	})
	return ok
}

func compareMetricLabelKeys(keys []string) descriptorComparator {
	lookup := make(map[string]bool)
	for _, key := range keys {
		lookup[key] = true
	}

	return func(t *testing.T, metric *pdata.Metric) bool {
		switch dataType := metric.DataType(); dataType {
		case pdata.MetricDataTypeGauge:
			dataPoints := metric.Gauge().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !ensureMatchingKeys(t, lookup, point.LabelsMap()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeHistogram:
			dataPoints := metric.Histogram().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !ensureMatchingKeys(t, lookup, point.LabelsMap()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeSummary:
			dataPoints := metric.Summary().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !ensureMatchingKeys(t, lookup, point.LabelsMap()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeSum:
			dataPoints := metric.Sum().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !ensureMatchingKeys(t, lookup, point.LabelsMap()) {
					return false
				}
			}
			return true

		default:
			t.Fatalf("Unhandled data type: %s", dataType)
			return false
		}
	}
}

func compareSeriesLabelValues(values []string) seriesComparator {
	lookup := make(map[string]bool)
	for _, value := range values {
		lookup[value] = true
	}

	return func(t *testing.T, metric *pdata.Metric) bool {
		switch dataType := metric.DataType(); dataType {
		case pdata.MetricDataTypeGauge:
			dataPoints := metric.Gauge().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !ensureMatchingValues(t, lookup, point.LabelsMap()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeHistogram:
			dataPoints := metric.Histogram().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !ensureMatchingValues(t, lookup, point.LabelsMap()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeSummary:
			dataPoints := metric.Summary().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !ensureMatchingValues(t, lookup, point.LabelsMap()) {
					return false
				}
			}
			return true

		case pdata.MetricDataTypeSum:
			dataPoints := metric.Sum().DataPoints()
			for k := 0; k < dataPoints.Len(); k++ {
				point := dataPoints.At(k)
				if !ensureMatchingValues(t, lookup, point.LabelsMap()) {
					return false
				}
			}
			return true

		default:
			t.Fatalf("Unhandled data type: %s", dataType)
			return false
		}
	}
}

/*
func doCompare(name string, t *testing.T, want, got *agentmetricspb.ExportMetricsServiceRequest, expectations []testExpectation) {
	t.Run(name, func(t *testing.T) {
		numScrapeMetrics := countScrapeMetrics(got)
		assert.Equal(t, expectedScrapeMetricCount, numScrapeMetrics)
		assert.EqualValues(t, want.Node, got.Node)
		assert.EqualValues(t, want.Resource, got.Resource)
		for _, e := range expectations {
			assert.True(t, e(t, got.Metrics))
		}
	})
}

func comparePointTimestamp(ts *timestamppb.Timestamp) pointComparator {
	return func(t *testing.T, point *metricspb.Point) bool {
		return assert.Equal(t, ts.String(), point.Timestamp.String())
	}
}

func compareDoubleVal(cmp float64) pointComparator {
	return func(t *testing.T, pt *metricspb.Point) bool {
		return assert.Equal(t, cmp, pt.GetDoubleValue())
	}
}

func compareHistogram(count int64, sum float64, buckets []int64) pointComparator {
	return func(t *testing.T, pt *metricspb.Point) bool {
		ret := assert.Equal(t, count, pt.GetDistributionValue().Count)
		ret = ret && assert.Equal(t, sum, pt.GetDistributionValue().Sum)

		if ret {
			for i, b := range buckets {
				ret = ret && assert.Equal(t, b, pt.GetDistributionValue().Buckets[i].Count)
			}
		}
		return ret
	}
}

func compareSummary(count int64, sum float64, quantiles map[float64]float64) pointComparator {
	return func(t *testing.T, pt *metricspb.Point) bool {
		ret := assert.Equal(t, count, pt.GetSummaryValue().Count.Value)
		ret = ret && assert.Equal(t, sum, pt.GetSummaryValue().Sum.Value)

		if ret {
			assert.Equal(t, len(quantiles), len(pt.GetSummaryValue().Snapshot.PercentileValues))
			for _, q := range pt.GetSummaryValue().Snapshot.PercentileValues {
				assert.Equal(t, quantiles[q.Percentile], q.Value)
			}
		}
		return ret
	}
}
*/

// Test data and validation functions for EndToEnd test
// Make sure every page has a gauge, we are relying on it to figure out the start time if needed

// target1 has one gauge, two counts of a same family, one histogram and one summary. We are expecting the both
// successful scrapes will produce all metrics using the first scrape's timestamp as start time.
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

func verifyTarget1(t *testing.T, td *testData, mds []*pdata.MetricSlice) {
	// TODO: Translate me.
	/*
		verifyNumScrapeResults(t, td, mds)
		if len(mds) < 1 {
			t.Fatal("At least one metric request should be present")
		}
		m1 := mds[0]
		// m1 has 4 metrics + 5 internal scraper metrics
		if l := len(m1.InstrumentationLibraryMetrics().Len()); l != 9 {
			t.Errorf("want 9, but got %v\n", l)
		}

		ts1 := m1.Metrics[0].Timeseries[0].Points[0].Timestamp
		e1 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareDoubleVal(19),
						},
					},
				}),
			assertMetricPresent("http_requests_total",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE),
					compareMetricLabelKeys([]string{"code", "method"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"200", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareDoubleVal(100),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"400", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareDoubleVal(5),
						},
					},
				}),
			assertMetricPresent("http_request_duration_seconds",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareHistogram(2500, 5000, []int64{1000, 500, 500, 500}),
						},
					},
				}),
			assertMetricPresent("rpc_duration_seconds",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_SUMMARY),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareSummary(1000, 5000, map[float64]float64{1: 1, 90: 5, 99: 8}),
						},
					},
				}),
		}

		want1 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
		}

		doCompare("scrape1", t, want1, m1, e1)

		// verify the 2nd metricData
		m2 := mds[1]
		ts2 := m2.Metrics[0].Timeseries[0].Points[0].Timestamp

		want2 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
		}

		e2 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareDoubleVal(18),
						},
					},
				}),
			assertMetricPresent("http_requests_total",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE),
					compareMetricLabelKeys([]string{"code", "method"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"200", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareDoubleVal(199),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"400", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareDoubleVal(12),
						},
					},
				}),
			assertMetricPresent("http_request_duration_seconds",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareHistogram(2600, 5050, []int64{1100, 500, 500, 500}),
						},
					},
				}),
			assertMetricPresent("rpc_duration_seconds",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_SUMMARY),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareSummary(1001, 5002, map[float64]float64{1: 1, 90: 6, 99: 8}),
						},
					},
				}),
		}

		doCompare("scrape2", t, want2, m2, e2)
	*/
}

// target2 is going to have 5 pages, and there's a newly added item on the 2nd page.
// with the 4th page, we are simulating a reset (values smaller than previous), start times should be from
// this run for the 4th and 5th scrapes.
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

func verifyTarget2(t *testing.T, td *testData, mds []*pdata.MetricSlice) {
	// TODO: Translate me.
	/*
		verifyNumScrapeResults(t, td, mds)
		m1 := mds[0]
		// m1 has 2 metrics + 5 internal scraper metrics
		if l := len(m1.Metrics); l != 7 {
			t.Errorf("want 7, but got %v\n", l)
		}

		ts1 := m1.Metrics[0].Timeseries[0].Points[0].Timestamp
		want1 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
		}

		e1 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareDoubleVal(18),
						},
					},
				}),
			assertMetricPresent("http_requests_total",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE),
					compareMetricLabelKeys([]string{"code", "method"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"200", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareDoubleVal(10),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"400", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareDoubleVal(50),
						},
					},
				}),
		}

		doCompare("scrape1", t, want1, m1, e1)

		// verify the 2nd metricData
		m2 := mds[1]
		ts2 := m2.Metrics[0].Timeseries[0].Points[0].Timestamp

		want2 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
		}
		e2 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareDoubleVal(16),
						},
					},
				}),
			assertMetricPresent("http_requests_total",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE),
					compareMetricLabelKeys([]string{"code", "method"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"200", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareDoubleVal(50),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"400", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareDoubleVal(60),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts2),
							compareSeriesLabelValues([]string{"500", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareDoubleVal(3),
						},
					},
				}),
		}

		doCompare("scrape2", t, want2, m2, e2)

		// verify the 3rd metricData, with the new code=500 counter which first appeared on 2nd run
		m3 := mds[2]
		// its start timestamp shall be from the 2nd run
		ts3 := m3.Metrics[0].Timeseries[0].Points[0].Timestamp

		want3 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
		}

		e3 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts3),
							compareDoubleVal(16),
						},
					},
				}),
			assertMetricPresent("http_requests_total",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE),
					compareMetricLabelKeys([]string{"code", "method"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"200", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts3),
							compareDoubleVal(50),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"400", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts3),
							compareDoubleVal(60),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts2),
							compareSeriesLabelValues([]string{"500", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts3),
							compareDoubleVal(5),
						},
					},
				}),
		}

		doCompare("scrape3", t, want3, m3, e3)

		// verify the 4th metricData which reset happens
		m4 := mds[3]
		ts4 := m4.Metrics[0].Timeseries[0].Points[0].Timestamp

		want4 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
			Metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:        "up",
						Description: "The scraping was successful",
						Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
						Unit:        "bool",
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							Points: []*metricspb.Point{
								{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
					},
				},
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:        "scrape_series_added",
						Description: "The approximate number of new series in this scrape",
						Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
						Unit:        "count",
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							Points: []*metricspb.Point{
								{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 14.0}},
							},
						},
					},
				},
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:        "scrape_duration_seconds",
						Description: "Duration of the scrape",
						Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
						Unit:        "seconds",
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							Points: []*metricspb.Point{
								{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 0.0123456}},
							},
						},
					},
				},
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:        "scrape_samples_scraped",
						Description: "The number of samples the target exposed",
						Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
						Unit:        "count",
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							Points: []*metricspb.Point{
								{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 14.0}},
							},
						},
					},
				},
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:        "scrape_samples_post_metric_relabeling",
						Description: "The number of samples remaining after metric relabeling was applied",
						Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
						Unit:        "count",
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							Points: []*metricspb.Point{
								{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 14.0}},
							},
						},
					},
				},
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:        "scrape_series_added",
						Description: "The approximate number of new series in this scrape",
						Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
						Unit:        "count",
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							Points: []*metricspb.Point{
								{Timestamp: ts2, Value: &metricspb.Point_DoubleValue{DoubleValue: 14.0}},
							},
						},
					},
				},
			},
		}

		e4 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts4),
							compareDoubleVal(16),
						},
					},
				}),
			assertMetricPresent("http_requests_total",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE),
					compareMetricLabelKeys([]string{"code", "method"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts4),
							compareSeriesLabelValues([]string{"200", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts4),
							compareDoubleVal(49),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts4),
							compareSeriesLabelValues([]string{"400", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts4),
							compareDoubleVal(59),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts4),
							compareSeriesLabelValues([]string{"500", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts4),
							compareDoubleVal(3),
						},
					},
				}),
		}

		doCompare("scrape4", t, want4, m4, e4)

		// verify the 5th metricData which reset happens
		m5 := mds[4]
		// its start timestamp shall be from the 4th run
		ts5 := m5.Metrics[0].Timeseries[0].Points[0].Timestamp

		want5 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
		}

		e5 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts5),
							compareDoubleVal(16),
						},
					},
				}),
			assertMetricPresent("http_requests_total",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE),
					compareMetricLabelKeys([]string{"code", "method"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts4),
							compareSeriesLabelValues([]string{"200", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts5),
							compareDoubleVal(50),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts4),
							compareSeriesLabelValues([]string{"400", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts5),
							compareDoubleVal(59),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts4),
							compareSeriesLabelValues([]string{"500", "post"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts5),
							compareDoubleVal(5),
						},
					},
				}),
		}

		doCompare("scrape5", t, want5, m5, e5)
	*/
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

func verifyTarget3(t *testing.T, td *testData, mds []*pdata.MetricSlice) {
	// TODO: Translate me.
	/*
		verifyNumScrapeResults(t, td, mds)
		m1 := mds[0]
		// m1 has 3 metrics + 5 internal scraper metrics
		if l := len(m1.Metrics); l != 8 {
			t.Errorf("want 8, but got %v\n", l)
		}

		ts1 := m1.Metrics[1].Timeseries[0].Points[0].Timestamp
		want1 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
		}

		e1 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareDoubleVal(18),
						},
					},
				}),
			assertMetricPresent("http_request_duration_seconds",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareHistogram(13003, 50000, []int64{10000, 1000, 1001, 1002}),
						},
					},
				}),
			assertMetricAbsent("corrupted_hist"),
			assertMetricPresent("rpc_duration_seconds",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_SUMMARY),
					compareMetricLabelKeys([]string{"foo"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"bar"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareSummary(900, 8000, map[float64]float64{1: 31, 5: 35, 50: 47, 90: 70, 99: 76}),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"no_quantile"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts1),
							compareSummary(50, 100, map[float64]float64{}),
						},
					},
				}),
		}

		doCompare("scrape1", t, want1, m1, e1)

		// verify the 2nd metricData
		m2 := mds[1]
		ts2 := m2.Metrics[0].Timeseries[0].Points[0].Timestamp

		want2 := &agentmetricspb.ExportMetricsServiceRequest{
			Node:     td.node,
			Resource: td.resource,
		}

		e2 := []testExpectation{
			assertMetricPresent("go_threads",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_GAUGE_DOUBLE),
				},
				[]seriesExpectation{
					{
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareDoubleVal(16),
						},
					},
				}),
			assertMetricPresent("http_request_duration_seconds",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareHistogram(14003, 50100, []int64{11000, 1000, 1001, 1002}),
						},
					},
				}),
			assertMetricAbsent("corrupted_hist"),
			assertMetricPresent("rpc_duration_seconds",
				[]descriptorComparator{
					compareMetricType(metricspb.MetricDescriptor_SUMMARY),
					compareMetricLabelKeys([]string{"foo"}),
				},
				[]seriesExpectation{
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"bar"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareSummary(950, 8100, map[float64]float64{1: 32, 5: 35, 50: 47, 90: 70, 99: 77}),
						},
					},
					{
						series: []seriesComparator{
							compareStartTimestamp(ts1),
							compareSeriesLabelValues([]string{"no_quantile"}),
						},
						points: []pointComparator{
							comparePointTimestamp(ts2),
							compareSummary(55, 101, map[float64]float64{}),
						},
					},
				}),
		}

		doCompare("scrape2", t, want2, m2, e2)
	*/
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

var startTimeMetricPageStartTimestamp = &timestamppb.Timestamp{Seconds: 971, Nanos: 800000000}

// 5 scraped metrics + 5 internal metrics where:
// 5 scraped metrics:
//   * go_threads
//   * http_requests_total
//   * http_request_duration_seconds
//   * rpc_duration_seconds
//   * process_start_time
//
// 5 internal metrics:
//   * scrape_duration_seconds
//   * scrape_samples_post_metric_relabeling
//   * scrape_samples_scraped
//   * scrape_series_added
//   * up
const numStartTimeMetricPageTimeseries = 10

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
	rcvr := newPrometheusReceiver(logger, &Config{
		ReceiverSettings:   config.NewReceiverSettings(config.NewID(typeStr)),
		PrometheusConfig:   cfg,
		UseStartTimeMetric: useStartTimeMetric}, cms)

	require.NoError(t, rcvr.Start(context.Background(), componenttest.NewNopHost()), "Failed to invoke Start: %v", err)
	t.Cleanup(func() {
		// verify state after shutdown is called
		assert.Lenf(t, flattenTargets(rcvr.scrapeManager.TargetsAll()), len(targets), "expected %v targets to be running", len(targets))
		require.NoError(t, rcvr.Shutdown(context.Background()))
		assert.Len(t, flattenTargets(rcvr.scrapeManager.TargetsAll()), 0, "expected scrape manager to have no targets")
	})

	// wait for all provided data to be scraped
	mp.wg.Wait()
	metrics := cms.AllMetrics()

	// split and store results by target name
	results := make(map[string][]*pdata.ResourceMetrics)
	for _, md := range metrics {
		rms := md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			rmi := rms.At(i)
			serviceNameAttr, ok := rmi.Resource().Attributes().Get("service.name")
			assert.True(t, ok, `expected "service.name" as a known attribute`)
			serviceName := serviceNameAttr.StringVal()
			results[serviceName] = append(results[serviceName], &rmi)
		}
	}

	lres, lep := len(results), len(mp.endpoints)
	assert.Equalf(t, lep, lres, "want %d targets, but got %v\n", lep, lres)

	// Skipping the validate loop below, because it falsely assumed that
	// staleness markers would not be returned, yet the tests are a bit rigid.
	if false {
		t.Log(`Skipping the "up" metric checks as they seem to be spuriously failing after staleness marker insertions`)
		return
	}

	// loop to validate outputs for each targets
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			mds := getValidScrapes(t, results[target.name])
			target.validateFunc(t, target, mds)
		})
	}
}

// flattenTargets takes a map of jobs to target and flattens to a list of targets
func flattenTargets(targets map[string][]*scrape.Target) []*scrape.Target {
	var flatTargets []*scrape.Target
	for _, target := range targets {
		flatTargets = append(flatTargets, target...)
	}
	return flatTargets
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

	// Splitting out targets, because the prior tests were oblivious
	// about staleness metrics being emitted, and hence when trying
	// to compare values across 2 different scrapes emits staleness
	// markers whose NaN values are unaccounted for.
	// TODO: Perhaps refactor these tests.
	for _, target := range targets {
		testEndToEndRegex(t, []*testData{target}, true, "^(.+_)*process_start_time_seconds$")
	}
}

func verifyStartTimeMetricPage(t *testing.T, td *testData, mds []*pdata.MetricSlice) {
	numMetrics := 0
	for _, metrics := range mds {
		for i := 0; i < metrics.Len(); i++ {
			metric := metrics.At(i)
			timestamp := startTimeMetricPageStartTimestamp
			numMetrics++
			switch dataType := metric.DataType(); dataType {
			case pdata.MetricDataTypeGauge:
				timestamp = nil
				dataPoints := metric.Gauge().DataPoints()
				for k := 0; k < dataPoints.Len(); k++ {
					point := dataPoints.At(k)
					assert.Equal(t, timestamp.AsTime(), point.StartTimestamp().AsTime(), metric.Name()+" :: "+metric.DataType().String())
				}

			case pdata.MetricDataTypeHistogram:
				timestamp = nil
				dataPoints := metric.Histogram().DataPoints()
				for k := 0; k < dataPoints.Len(); k++ {
					point := dataPoints.At(k)
					assert.Equal(t, timestamp.AsTime(), point.StartTimestamp().AsTime(), metric.Name()+" :: "+metric.DataType().String())
				}

			case pdata.MetricDataTypeSummary:
				dataPoints := metric.Summary().DataPoints()
				for k := 0; k < dataPoints.Len(); k++ {
					point := dataPoints.At(k)
					assert.Equal(t, timestamp.AsTime(), point.StartTimestamp().AsTime(), metric.Name()+" :: "+metric.DataType().String())
				}

			case pdata.MetricDataTypeSum:
				timestamp = nil
				dataPoints := metric.Sum().DataPoints()
				for k := 0; k < dataPoints.Len(); k++ {
					point := dataPoints.At(k)
					assert.Equal(t, timestamp.AsTime(), point.StartTimestamp().AsTime(), metric.Name()+" :: "+metric.DataType().String())
				}

			default:
				t.Errorf("Unhandled data type: %s", dataType)
			}
		}
	}
	assert.Equal(t, numStartTimeMetricPageTimeseries, numMetrics)
}

func testEndToEndRegex(t *testing.T, targets []*testData, useStartTimeMetric bool, startTimeMetricRegex string) {
	// 1. setup mock server
	mp, cfg, err := setupMockPrometheus(targets...)
	require.Nilf(t, err, "Failed to create Promtheus config: %v", err)
	defer mp.Close()

	cms := new(consumertest.MetricsSink)
	rcvr := newPrometheusReceiver(logger, &Config{
		ReceiverSettings:     config.NewReceiverSettings(config.NewID(typeStr)),
		PrometheusConfig:     cfg,
		UseStartTimeMetric:   useStartTimeMetric,
		StartTimeMetricRegex: startTimeMetricRegex}, cms)

	require.NoError(t, rcvr.Start(context.Background(), componenttest.NewNopHost()), "Failed to invoke Start: %v", err)
	t.Cleanup(func() { require.NoError(t, rcvr.Shutdown(context.Background())) })

	// wait for all provided data to be scraped
	mp.wg.Wait()
	metrics := cms.AllMetrics()

	// split and store results by target name
	results := make(map[string][]*pdata.MetricSlice)
	for _, md := range metrics {
		rms := md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			rmi := rms.At(i)
			serviceNameAttr, ok := rmi.Resource().Attributes().Get("service.name")
			assert.True(t, ok, `expected "service.name" as a known attribute`)
			serviceName := serviceNameAttr.StringVal()
			ilmL := rmi.InstrumentationLibraryMetrics()
			for j := 0; j < ilmL.Len(); j++ {
				ilm := ilmL.At(j)
				metricL := ilm.Metrics()
				results[serviceName] = append(results[serviceName], &metricL)
			}
		}
	}

	lres, lep := len(results), len(mp.endpoints)
	assert.Equalf(t, lep, lres, "want %d targets, but got %v\n", lep, lres)

	// loop to validate outputs for each targets
	for _, target := range targets {
		target.validateFunc(t, target, results[target.name])
	}
}
