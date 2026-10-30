// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pipeline"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
)

// expensiveRequest simulates tracesRequest where ItemsCount walks 2k spans.
type expensiveRequest struct {
	itemsCalls int
}

func (r *expensiveRequest) ItemsCount() int {
	r.itemsCalls++
	// Simulate SpanCount walk over 2k spans.
	n := 0
	for i := 0; i < 2000; i++ {
		n++
	}
	return n
}

func (r *expensiveRequest) MergeSplit(context.Context, int, request.SizerType, request.Request) ([]request.Request, error) {
	return []request.Request{r}, nil
}

func (r *expensiveRequest) BytesSize() int { return 2000 }

func newNopObsQueueBench(b *testing.B, delegate Queue[request.Request]) Queue[request.Request] {
	b.Helper()
	set := Settings[request.Request]{
		Signal:    pipeline.SignalTraces,
		ID:        exporterID,
		Telemetry: componenttest.NewNopTelemetrySettings(),
		SizerType: request.SizerTypeItems,
		Capacity:  1000000,
	}
	// Nop ObsMetrics: both funcs nil, like NewExporterObsMetrics failure path or empty.
	nopMetrics := queuebatchtelemetry.ObsMetrics{}
	q, err := newObsQueue(set, nopMetrics, delegate)
	if err != nil {
		b.Fatal(err)
	}
	return q
}

func BenchmarkObsQueueNopExpensive(b *testing.B) {
	delegate := newFakeQueue[request.Request](nil, 0, 1000000)
	q := newNopObsQueueBench(b, delegate)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := &expensiveRequest{}
		_ = q.Offer(context.Background(), r)
		if r.itemsCalls != 0 {
			b.Fatalf("ItemsCount called %d times with nop telemetry, want 0", r.itemsCalls)
		}
	}
}
