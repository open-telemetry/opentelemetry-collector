// Copyright 2018, OpenCensus Authors
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

package ocinterceptor_test

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/census-instrumentation/opencensus-service/interceptor/opencensus"
	"github.com/census-instrumentation/opencensus-service/internal"
)

// Ensure that if we add a metrics exporter that our target metrics
// will be recorded but also with the proper tag keys and values.
// See Issue https://github.com/census-instrumentation/opencensus-service/issues/63
//
// Note: we are intentionally skipping the ocgrpc.ServerDefaultViews as this
// test is to ensure exactness, but with the mentioned views registered, the
// output will be quite noisy.
func TestEnsureRecordedMetrics(t *testing.T) {
	sappender := newSpanAppender()

	_, port, doneFn := ocInterceptorOnGRPCServer(t, sappender, ocinterceptor.WithSpanBufferPeriod(2*time.Millisecond))
	defer doneFn()

	// Now the opencensus-agent exporter.
	oce, err := ocagent.NewExporter(ocagent.WithPort(uint16(port)), ocagent.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to create the ocagent-exporter: %v", err)
	}
	trace.RegisterExporter(oce)
	defer func() {
		oce.Stop()
		trace.UnregisterExporter(oce)
	}()

	// Now for the stats exporter
	if err := view.Register(internal.AllViews...); err != nil {
		t.Fatalf("Failed to register all views: %v", err)
	}
	defer view.Unregister(internal.AllViews...)

	metricsReportingPeriod := 5 * time.Millisecond
	view.SetReportingPeriod(metricsReportingPeriod)
	// On exit, revert the metrics reporting period.
	defer view.SetReportingPeriod(60 * time.Second)

	cme := newCountMetricsExporter()
	view.RegisterExporter(cme)
	defer view.UnregisterExporter(cme)

	n := 20
	// Now it is time to send over some spans
	// and we'll count the numbers received.
	for i := 0; i < n; i++ {
		now := time.Now().UTC()
		oce.ExportSpan(&trace.SpanData{
			StartTime: now.Add(-10 * time.Second),
			EndTime:   now.Add(20 * time.Second),
			SpanContext: trace.SpanContext{
				TraceID:      trace.TraceID{byte(0x20 + i), 0x4E, 0x4D, 0x4C, 0x4B, 0x4A, 0x49, 0x48, 0x47, 0x46, 0x45, 0x44, 0x43, 0x42, 0x41},
				SpanID:       trace.SpanID{0x7F, 0x7E, 0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78},
				TraceOptions: trace.TraceOptions(i & 0x01),
			},
			ParentSpanID: trace.SpanID{byte(0x01 + i), 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37},
			Name:         fmt.Sprintf("Span-%d", i),
			Status:       trace.Status{Code: trace.StatusCodeInternal, Message: "Blocked by firewall"},
		})
	}

	// Give them some time to be exported.
	// say n * metricsReportingPeriod
	<-time.After(time.Duration(n) * metricsReportingPeriod)
	oce.Flush()

	checkCountMetricsExporterResults(t, cme, n, 1)
}

func TestEnsureRecordedMetrics_zeroLengthSpansSender(t *testing.T) {
	t.Skipf("Currently disabled, enable this test when the following are fixed:\nIssue %s\nPR %s",
		"https://github.com/census-instrumentation/opencensus-go/issues/862",
		"https://github.com/census-instrumentation/opencensus-go/pull/922",
	)
	sappender := newSpanAppender()

	_, port, doneFn := ocInterceptorOnGRPCServer(t, sappender, ocinterceptor.WithSpanBufferPeriod(2*time.Millisecond))
	defer doneFn()

	// Now the opencensus-agent exporter.
	oce, err := ocagent.NewExporter(ocagent.WithPort(uint16(port)), ocagent.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to create the ocagent-exporter: %v", err)
	}
	trace.RegisterExporter(oce)
	defer func() {
		oce.Stop()
		trace.UnregisterExporter(oce)
	}()

	// Now for the stats exporter
	if err := view.Register(internal.AllViews...); err != nil {
		t.Fatalf("Failed to register all views: %v", err)
	}
	defer view.Unregister(internal.AllViews...)

	metricsReportingPeriod := 10 * time.Millisecond
	view.SetReportingPeriod(metricsReportingPeriod)
	// On exit, revert the metrics reporting period.
	defer view.SetReportingPeriod(60 * time.Second)

	cme := newCountMetricsExporter()
	view.RegisterExporter(cme)
	defer view.UnregisterExporter(cme)

	n := 20
	// Now for the traceExporter that sends 0 length spans
	traceSvcClient, traceSvcDoneFn, err := makeTraceServiceClient(port)
	if err != nil {
		t.Fatalf("Failed to create the trace service client: %v", err)
	}
	defer traceSvcDoneFn()
	for i := 0; i <= n; i++ {
		_ = traceSvcClient.Send(&agenttracepb.ExportTraceServiceRequest{Spans: nil, Node: &commonpb.Node{}})
	}
	<-time.After(time.Duration(n) * metricsReportingPeriod)
	checkCountMetricsExporterResults(t, cme, n, 0)
}

func checkCountMetricsExporterResults(t *testing.T, cme *countMetricsExporter, n int, wantAllCountsToBe int64) {
	cme.mu.Lock()
	defer cme.mu.Unlock()

	// The only tags that we are expecting are "opencensus_interceptor": "opencensus" * n
	wantTagKey, _ := tag.NewKey("opencensus_interceptor")
	valuesPlusBlank := strings.Split(strings.Repeat("opencensus,opencensus,", n/2), ",")
	wantValues := valuesPlusBlank[:len(valuesPlusBlank)-1]
	wantTags := map[tag.Key][]string{
		wantTagKey: wantValues,
	}

	gotTags := cme.tags
	if !reflect.DeepEqual(gotTags, wantTags) {
		t.Errorf("\nGotTags:\n\t%#v\n\nWantTags:\n\t%#v\n", gotTags, wantTags)
	}

	// The only data types we are expecting are:
	// * DistributionData
	for key, aggregation := range cme.data {
		switch agg := aggregation.(type) {
		case *view.DistributionData:
			if g, w := agg.Count, int64(1); g != w {
				t.Errorf("Data point #%d GotCount %d Want %d", key, g, w)
			}
		default:
			t.Errorf("Data point #%d Got %T want %T", key, agg, (*view.DistributionData)(nil))
		}
	}
}

type countMetricsExporter struct {
	mu   sync.Mutex
	tags map[tag.Key][]string
	data map[int]view.AggregationData
}

func newCountMetricsExporter() *countMetricsExporter {
	return &countMetricsExporter{
		tags: make(map[tag.Key][]string),
		data: make(map[int]view.AggregationData),
	}
}

func (cme *countMetricsExporter) clear() {
	cme.mu.Lock()
	defer cme.mu.Unlock()

	cme.data = make(map[int]view.AggregationData)
	cme.tags = make(map[tag.Key][]string)
}

var _ view.Exporter = (*countMetricsExporter)(nil)

func (cme *countMetricsExporter) ExportView(vd *view.Data) {
	cme.mu.Lock()
	defer cme.mu.Unlock()

	for _, row := range vd.Rows {
		cme.data[len(cme.data)] = row.Data
		for _, tag_ := range row.Tags {
			cme.tags[tag_.Key] = append(cme.tags[tag_.Key], tag_.Value)
		}
	}
}
