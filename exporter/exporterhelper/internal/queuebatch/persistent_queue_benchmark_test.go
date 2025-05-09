// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"encoding/json"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

type requestTypeKeyType struct{}

var requestTypeKey = requestTypeKeyType{}

const (
	originalRequestValue           = "original_request"
	contextRequestValue            = "context_request"
	contextAndKeyValueRequestValue = "context_and_key_value_request"
)

type largeRequest struct {
	ID   int
	Meta MetaInfo
	Body BodyLevel1
}

type MetaInfo struct {
	Source string
	Tags   []string
}

type BodyLevel1 struct {
	Level2 BodyLevel2
	Extra  string
}

type BodyLevel2 struct {
	Level3 BodyLevel3
	Values []int
}

type BodyLevel3 struct {
	Payload []byte
	Note    string
}

type largeRequestEncoding struct{}

func (largeRequestEncoding) Marshal(val largeRequest) ([]byte, error) {
	return json.Marshal(val)
}

func (largeRequestEncoding) Unmarshal(buf []byte) (largeRequest, error) {
	var req largeRequest
	if err := json.Unmarshal(buf, &req); err != nil {
		return largeRequest{}, err
	}
	return req, nil
}

func BenchmarkPersistentQueue_LargeRequests(b *testing.B) {
	const (
		numRequests = 3000
		dataSize    = 20 * 1024 // 20 KB
	)
	// Prepare large requests
	requests := make([]largeRequest, numRequests)
	for i := range requests {
		requests[i] = largeRequest{
			ID: i,
			Meta: MetaInfo{
				Source: "benchmark",
				Tags:   []string{"tag1", "tag2", "tag3"},
			},
			Body: BodyLevel1{
				Level2: BodyLevel2{
					Level3: BodyLevel3{
						Payload: make([]byte, dataSize),
						Note:    "deep payload",
					},
					Values: []int{i, i + 1, i + 2},
				},
				Extra: "extra info",
			},
		}
		for j := range requests[i].Body.Level2.Level3.Payload {
			requests[i].Body.Level2.Level3.Payload[j] = byte((i + j) % 256)
		}
	}

	// Use a persistent queue with large capacity
	pq := newPersistentQueue[largeRequest](persistentQueueSettings[largeRequest]{
		sizer:     request.RequestsSizer[largeRequest]{},
		capacity:  numRequests,
		signal:    pipeline.SignalTraces,
		storageID: component.ID{},
		encoding:  largeRequestEncoding{},
		id:        component.NewID(exportertest.NopType),
		telemetry: componenttest.NewNopTelemetrySettings(),
	}).(*persistentQueue[largeRequest])

	ext := storagetest.NewMockStorageExtension(nil)
	client, err := ext.GetClient(context.Background(), component.KindExporter, pq.set.id, pq.set.signal.String())
	if err != nil {
		b.Fatalf("failed to get storage client: %v", err)
	}

	contextValues := []string{originalRequestValue, contextRequestValue} // , contextAndKeyValueRequestValue}
	for _, value := range contextValues {
		contextType := context.WithValue(context.Background(), requestTypeKey, value)
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
			SpanID:     trace.SpanID{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
			TraceFlags: trace.TraceFlags(0x1),
			TraceState: trace.TraceState{},
			Remote:     false,
		})
		sharedContext := trace.ContextWithSpanContext(contextType, sc)
		pq.initClient(contextType, client)

		b.ResetTimer()
		b.ReportAllocs()

		// Offer all requests
		for i := 0; i < numRequests; i++ {
			err := pq.Offer(sharedContext, requests[i])
			if err != nil {
				b.Fatalf("Offer failed at %d: %v", i, err)
			}
		}

		// Read and OnDone all requests
		for i := 0; i < numRequests; i++ {
			_, req, done, ok := pq.Read(sharedContext)
			if !ok {
				b.Fatalf("Read failed at %d", i)
			}
			if req.ID != i {
				b.Fatalf("Request ID mismatch at %d: got %d", i, req.ID)
			}
			done.OnDone(nil)
		}

		if pq.Size() != 0 {
			b.Fatalf("Queue not empty after all operations: size=%d", pq.Size())
		}
	}
}
