// Copyright 2019, OpenTelemetry Authors
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

package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/config"
	sd_config "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/scrape"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/prometheus/prometheus/discovery"
)

func TestOcaStore(t *testing.T) {

	o := NewOcaStore(context.Background(), nil, nil)

	_, err := o.Appender()
	if err == nil {
		t.Fatal("expecting error, but get nil")
	}

	o.SetScrapeManager(nil)
	_, err = o.Appender()
	if err == nil {
		t.Fatal("expecting error, but get nil")
	}

	o.SetScrapeManager(&scrape.Manager{})

	app, err := o.Appender()
	if app == nil {
		t.Fatalf("expecting app, but got error %v\n", err)
	}

	_ = o.Close()

	app, err = o.Appender()
	if app == nil || err != nil {
		t.Fatalf("expect app!=nil and err==nil, got app=%v and err=%v", app, err)
	}

}

func TestOcaStoreIntegration(t *testing.T) {
	// verify at a high level
	type v struct {
		mname     string
		mtype     metricspb.MetricDescriptor_Type
		numLbKeys int
		numTs     int
	}
	tests := []struct {
		name string
		page string
		mv   []v
	}{
		{
			name: "promethues_text_format_example",
			page: testData1,
			mv: []v{
				{
					mname:     "http_requests_total",
					mtype:     metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
					numLbKeys: 2,
					numTs:     2,
				},
				{
					mname:     "http_request_duration_seconds",
					mtype:     metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
					numLbKeys: 0,
					numTs:     1,
				},
				{
					mname:     "rpc_duration_seconds",
					mtype:     metricspb.MetricDescriptor_SUMMARY,
					numLbKeys: 0,
					numTs:     1,
				},
			},
		},
		{
			name: "test_my_histogram_vec",
			page: testDataHistoVec,
			mv: []v{
				{
					mname:     "test_my_histogram_vec",
					mtype:     metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
					numLbKeys: 2,
					numTs:     2,
				},
			},
		},
		{
			name: "test_my_summary_vec",
			page: testDataSummaryVec,
			mv: []v{
				{
					mname:     "test_my_summary",
					mtype:     metricspb.MetricDescriptor_SUMMARY,
					numLbKeys: 0,
					numTs:     1,
				},
				{
					mname:     "test_my_summary_vec",
					mtype:     metricspb.MetricDescriptor_SUMMARY,
					numLbKeys: 2,
					numTs:     3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := startMockServer(tt.page)
			target, _ := url.Parse(srv.URL)
			defer srv.Close()
			mc, cancel := startScraper(target.Host)
			defer cancel()

			var d *consumerdata.MetricsData
			select {
			case d = <-mc.Metrics:
			case <-time.After(time.Second * 10):
				t.Error("no data come back in 10 second")
			}

			if len(tt.mv) != len(d.Metrics) {
				t.Errorf("Number of metrics got=%v exepct=%v\n", len(d.Metrics), len(tt.mv))
			}

			for i, dd := range d.Metrics {
				got := v{
					mname:     dd.MetricDescriptor.Name,
					mtype:     dd.MetricDescriptor.Type,
					numLbKeys: len(dd.MetricDescriptor.LabelKeys),
					numTs:     len(dd.Timeseries),
				}
				if !reflect.DeepEqual(got, tt.mv[i]) {
					t.Errorf("got %v, expect %v", got, tt.mv[i])
				}
			}
		})
	}

}

func startMockServer(page string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(page))
	}))
}

func startScraper(target string) (*mockConsumer, context.CancelFunc) {
	rawCfg := fmt.Sprintf(scrapCfgTpl, target)

	cfg, err := config.Load(rawCfg)
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	con := newMockConsumer()
	o := NewOcaStore(context.Background(), con, testLogger)
	scrapeManager := scrape.NewManager(log.NewNopLogger(), o)
	o.SetScrapeManager(scrapeManager)

	ctx := context.Background()
	ctxScrape, cancelScrape := context.WithCancel(ctx)

	discoveryManagerScrape := discovery.NewManager(ctxScrape, NewZapToGokitLogAdapter(zapLogger))
	go discoveryManagerScrape.Run()
	scrapeManager.ApplyConfig(cfg)

	// Run the scrape manager.
	syncConfig := make(chan bool)
	errsChan := make(chan error, 1)
	go func() {
		defer close(errsChan)

		<-time.After(100 * time.Millisecond)
		close(syncConfig)
		if err := scrapeManager.Run(discoveryManagerScrape.SyncCh()); err != nil {
			errsChan <- err
		}
	}()
	<-syncConfig
	// By this point we've given time to the scrape manager
	// to start applying its original configuration.

	discoveryCfg := make(map[string]sd_config.ServiceDiscoveryConfig)
	for _, scrapeConfig := range cfg.ScrapeConfigs {
		discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfig
	}

	// Now trigger the discovery notification to the scrape manager.
	discoveryManagerScrape.ApplyConfig(discoveryCfg)

	return con, cancelScrape
}

var scrapCfgTpl = `

scrape_configs:
- job_name: 'test'
  scrape_interval: 80ms
  static_configs:
  - targets: ['%s']
`

// https://github.com/prometheus/docs/blob/master/content/docs/instrumenting/exposition_formats.md#text-format-example
var testData1 = `
# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 1027 1395066363000
http_requests_total{method="post",code="400"}    3 1395066363000

# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 24054
http_request_duration_seconds_bucket{le="0.1"} 33444
http_request_duration_seconds_bucket{le="0.2"} 100392
http_request_duration_seconds_bucket{le="0.5"} 129389
http_request_duration_seconds_bucket{le="1"} 133988
http_request_duration_seconds_bucket{le="+Inf"} 144320
http_request_duration_seconds_sum 53423
http_request_duration_seconds_count 144320

# Finally a summary, which has a complex representation, too:
# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 3102
rpc_duration_seconds{quantile="0.05"} 3272
rpc_duration_seconds{quantile="0.5"} 4773
rpc_duration_seconds{quantile="0.9"} 9001
rpc_duration_seconds{quantile="0.99"} 76656
rpc_duration_seconds_sum 1.7560473e+07
rpc_duration_seconds_count 2693
`

var testDataHistoVec = `
# HELP test_my_histogram_vec This is my histogram vec
# TYPE test_my_histogram_vec histogram
test_my_histogram_vec_bucket{bar="",foo="f",le="0.0"} 0.0
test_my_histogram_vec_bucket{bar="",foo="f",le="30.0"} 2.0
test_my_histogram_vec_bucket{bar="",foo="f",le="50.0"} 5.0
test_my_histogram_vec_bucket{bar="",foo="f",le="80.0"} 8.0
test_my_histogram_vec_bucket{bar="",foo="f",le="100.0"} 9.0
test_my_histogram_vec_bucket{bar="",foo="f",le="200.0"} 9.0
test_my_histogram_vec_bucket{bar="",foo="f",le="+Inf"} 9.0
test_my_histogram_vec_sum{bar="",foo="f"} 420.06922114567107
test_my_histogram_vec_count{bar="",foo="f"} 9.0
test_my_histogram_vec_bucket{bar="b",foo="",le="0.0"} 0.0
test_my_histogram_vec_bucket{bar="b",foo="",le="30.0"} 4.0
test_my_histogram_vec_bucket{bar="b",foo="",le="50.0"} 6.0
test_my_histogram_vec_bucket{bar="b",foo="",le="80.0"} 9.0
test_my_histogram_vec_bucket{bar="b",foo="",le="100.0"} 9.0
test_my_histogram_vec_bucket{bar="b",foo="",le="200.0"} 9.0
test_my_histogram_vec_bucket{bar="b",foo="",le="+Inf"} 9.0
test_my_histogram_vec_sum{bar="b",foo=""} 307.6198217635357
test_my_histogram_vec_count{bar="b",foo=""} 9.0
`

var testDataSummaryVec = `
# HELP test_my_summary This is my summary
# TYPE test_my_summary summary
test_my_summary{quantile="0.5"} 5.297674177868265
test_my_summary{quantile="0.9"} 9.178352863969653
test_my_summary{quantile="0.99"} 9.178352863969653
test_my_summary_sum 41.730240267137724
test_my_summary_count 9.0
# HELP test_my_summary_vec This is my summary vec
# TYPE test_my_summary_vec summary
test_my_summary_vec{bar="b1",foo="f1",quantile="0.5"} 7.11378856098672
test_my_summary_vec{bar="b1",foo="f1",quantile="0.9"} 8.533665390719884
test_my_summary_vec{bar="b1",foo="f1",quantile="0.99"} 8.533665390719884
test_my_summary_vec_sum{bar="b1",foo="f1"} 55.1297982492356
test_my_summary_vec_count{bar="b1",foo="f1"} 9.0
test_my_summary_vec{bar="b3",foo="f1",quantile="0.5"} 3.430603231384155
test_my_summary_vec{bar="b3",foo="f1",quantile="0.9"} 9.938629091300923
test_my_summary_vec{bar="b3",foo="f1",quantile="0.99"} 9.938629091300923
test_my_summary_vec_sum{bar="b3",foo="f1"} 40.31948120896123
test_my_summary_vec_count{bar="b3",foo="f1"} 9.0
test_my_summary_vec{bar="b3",foo="f2",quantile="0.5"} 5.757833591018714
test_my_summary_vec{bar="b3",foo="f2",quantile="0.9"} 8.186181950691724
test_my_summary_vec{bar="b3",foo="f2",quantile="0.99"} 8.186181950691724
test_my_summary_vec_sum{bar="b3",foo="f2"} 46.82000375730371
test_my_summary_vec_count{bar="b3",foo="f2"} 9.0
`
