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
package exporterhelper

import (
	"context"
	"errors"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter"
)

// TODO https://github.com/open-telemetry/opentelemetry-service/issues/266
// Migrate tests to use testify/assert instead of t.Fatal pattern.
func TestMetricsExporter_InvalidName(t *testing.T) {
	if _, err := NewMetricsExporter("", newPushMetricsData(0, nil)); err != errEmptyExporterName {
		t.Fatalf("NewMetricsExporter returns: Want %v Got %v", errEmptyExporterName, err)
	}
}

func TestMetricsExporter_NilPushMetricsData(t *testing.T) {
	if _, err := NewMetricsExporter(fakeExporterName, nil); err != errNilPushMetricsData {
		t.Fatalf("NewMetricsExporter returns: Want %v Got %v", errNilPushMetricsData, err)
	}
}

func TestMetricsExporter_Default(t *testing.T) {
	md := consumerdata.MetricsData{}
	me, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, nil))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Nil(t, me.ConsumeMetricsData(context.Background(), md))

	assert.Equal(t, me.Name(), fakeExporterName)

	assert.Nil(t, me.Shutdown())
}

func TestMetricsExporter_Default_ReturnError(t *testing.T) {
	td := consumerdata.MetricsData{}
	want := errors.New("my_error")
	te, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, want))
	if err != nil {
		t.Fatalf("NewMetricsExporter returns: Want nil Got %v", err)
	}
	if err := te.ConsumeMetricsData(context.Background(), td); err != want {
		t.Fatalf("ConsumeMetricsData returns: Want %v Got %v", want, err)
	}
}

func TestMetricsExporter_WithSpan(t *testing.T) {
	te, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, nil), WithSpanName(fakeSpanName))
	if err != nil {
		t.Fatalf("NewMetricsExporter returns: Want nil Got %v", err)
	}
	checkWrapSpanForMetricsExporter(t, te, nil, 0)
}

func TestMetricsExporter_WithSpan_NonZeroDropped(t *testing.T) {
	te, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(1, nil), WithSpanName(fakeSpanName))
	if err != nil {
		t.Fatalf("NewMetricsExporter returns: Want nil Got %v", err)
	}
	checkWrapSpanForMetricsExporter(t, te, nil, 1)
}

func TestMetricsExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, want), WithSpanName(fakeSpanName))
	if err != nil {
		t.Fatalf("NewMetricsExporter returns: Want nil Got %v", err)
	}
	checkWrapSpanForMetricsExporter(t, te, want, 0)
}

func TestMetricsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func() error { shutdownCalled = true; return nil }

	me, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Nil(t, me.Shutdown())
	assert.True(t, shutdownCalled)
}

func TestMetricsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func() error { return want }

	me, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Equal(t, me.Shutdown(), want)
}

func newPushMetricsData(droppedSpans int, retError error) PushMetricsData {
	return func(ctx context.Context, td consumerdata.MetricsData) (int, error) {
		return droppedSpans, retError
	}
}

func generateMetricsTraffic(t *testing.T, te exporter.MetricsExporter, numRequests int, wantError error) {
	td := consumerdata.MetricsData{Metrics: make([]*metricspb.Metric, 1)}
	ctx, span := trace.StartSpan(context.Background(), fakeParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		if err := te.ConsumeMetricsData(ctx, td); err != wantError {
			t.Fatalf("Want %v Got %v", wantError, err)
		}
	}
}

func checkWrapSpanForMetricsExporter(t *testing.T, te exporter.MetricsExporter, wantError error, droppedSpans int) {
	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	const numRequests = 5
	generateMetricsTraffic(t, te, numRequests, wantError)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	if len(ocSpansSaver.spanData) == 0 {
		t.Fatal("No exported span data.")
	}

	gotSpanData := ocSpansSaver.spanData[:]
	if g, w := len(gotSpanData), numRequests+1; g != w {
		t.Fatalf("Spandata count: Want %d Got %d", w, g)
	}

	parentSpan := gotSpanData[numRequests]
	if g, w := parentSpan.Name, fakeParentSpanName; g != w {
		t.Fatalf("Parent span name: Want %s Got %s\nSpanData %v", w, g, parentSpan)
	}

	for _, sd := range gotSpanData[:numRequests] {
		if g, w := sd.ParentSpanID, parentSpan.SpanContext.SpanID; g != w {
			t.Fatalf("Exporter span not a child: Want %d Got %d\nSpanData %v", w, g, sd)
		}
		if g, w := sd.Status, errToStatus(wantError); g != w {
			t.Fatalf("Status: Want %v Got %v\nSpanData %v", w, g, sd)
		}
		if g, w := sd.Attributes[numReceivedMetricsAttribute], int64(1); g != w {
			t.Fatalf("Number of received spans attribute: Want %d Got %d\nSpanData %v", w, g, sd)
		}
		if g, w := sd.Attributes[numDroppedMetricsAttribute], int64(droppedSpans); g != w {
			t.Fatalf("Number of dropped spans attribute: Want %d Got %d\nSpanData %v", w, g, sd)
		}
	}
}
