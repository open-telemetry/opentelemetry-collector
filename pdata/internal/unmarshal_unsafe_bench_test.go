// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sync"
	"testing"

	"google.golang.org/grpc/mem"
)

const benchPayloadBytes = 10 << 20

var (
	benchTracePayload     []byte
	benchTracePayloadOnce sync.Once
	benchLogsPayload      []byte
	benchLogsPayloadOnce  sync.Once
)

func bench10MBTracePayload() []byte {
	benchTracePayloadOnce.Do(func() {
		benchTracePayload = genTracePayload(benchPayloadBytes)
	})
	return benchTracePayload
}

func genTracePayload(targetBytes int) []byte {
	req := NewExportTraceServiceRequest()
	rs := NewResourceSpans()
	rs.SchemaUrl = "https://opentelemetry.io/schemas/1.21.0"
	rs.Resource.Attributes = []KeyValue{
		stringAttr("service.name", "checkout-service"),
		stringAttr("service.version", "1.24.0"),
		stringAttr("deployment.environment", "production"),
	}

	ss := NewScopeSpans()
	ss.SchemaUrl = rs.SchemaUrl
	ss.Scope.Name = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	ss.Scope.Version = "1.24.0"

	attrKeys := []string{
		"http.method",
		"http.route",
		"http.url",
		"user_agent.original",
		"server.address",
		"url.path",
	}
	attrVals := []string{
		"GET",
		"/api/v1/users/{id}",
		"https://example.com/api/v1/users/12345?verbose=true&include=profile",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko)",
		"checkout.example.internal",
		"/api/v1/users/12345",
	}

	for i := 0; ; i++ {
		span := NewSpan()
		span.Name = "GET /api/v1/users/{id}"
		span.TraceState = "rojo=00f067aa0ba902b7,congo=t61rcWkgMzE"
		span.StartTimeUnixNano = 1700000000000000000
		span.EndTimeUnixNano = 1700000000001000000
		span.TraceId = *GenTestTraceID()
		span.SpanId = *GenTestSpanID()
		span.ParentSpanId = *GenTestSpanID()
		span.Attributes = make([]KeyValue, len(attrKeys))
		for k := range attrKeys {
			span.Attributes[k] = stringAttr(attrKeys[k], attrVals[k])
		}
		ss.Spans = append(ss.Spans, span)
		if i%64 != 63 {
			continue
		}
		rs.ScopeSpans = []*ScopeSpans{ss}
		req.ResourceSpans = []*ResourceSpans{rs}
		if req.SizeProto() >= targetBytes {
			break
		}
	}

	buf := make([]byte, req.SizeProto())
	req.MarshalProto(buf)
	return buf
}

func stringAttr(key, value string) KeyValue {
	return KeyValue{
		Key: key,
		Value: AnyValue{
			Value: &AnyValue_StringValue{StringValue: value},
		},
	}
}

func bench10MBLogsPayload() []byte {
	benchLogsPayloadOnce.Do(func() {
		benchLogsPayload = genLogsPayload(benchPayloadBytes)
	})
	return benchLogsPayload
}

func genLogsPayload(targetBytes int) []byte {
	req := NewExportLogsServiceRequest()
	rl := NewResourceLogs()
	rl.SchemaUrl = "https://opentelemetry.io/schemas/1.21.0"
	rl.Resource.Attributes = []KeyValue{
		stringAttr("service.name", "checkout-service"),
		stringAttr("service.version", "1.24.0"),
		stringAttr("deployment.environment", "production"),
		stringAttr("k8s.pod.name", "checkout-service-7d9c4b8f6-xk2nq"),
	}

	sl := NewScopeLogs()
	sl.SchemaUrl = rl.SchemaUrl
	sl.Scope.Name = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sl.Scope.Version = "1.24.0"

	const body = `failed to fetch user profile from cache, falling back to database: context deadline exceeded after 200ms method=GET path=/api/v1/users/12345 request_id=7f3a9c2e-4b1d-4e8a-9c0f-12ab34cd56ef peer=10.0.12.34`

	attrKeys := []string{
		"http.method",
		"http.route",
		"http.url",
		"user_agent.original",
		"k8s.namespace.name",
		"log.file.path",
	}
	attrVals := []string{
		"GET",
		"/api/v1/users/{id}",
		"https://example.com/api/v1/users/12345?verbose=true&include=profile",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko)",
		"checkout",
		"/var/log/pods/checkout_checkout-service-7d9c4b8f6-xk2nq_1/app/0.log",
	}

	for i := 0; ; i++ {
		lr := NewLogRecord()
		lr.TimeUnixNano = 1700000000000000000
		lr.ObservedTimeUnixNano = 1700000000001000000
		lr.SeverityNumber = SeverityNumber(9)
		lr.SeverityText = "INFO"
		lr.Body.Value = &AnyValue_StringValue{StringValue: body}
		lr.EventName = "user.profile.fetch"
		lr.TraceId = *GenTestTraceID()
		lr.SpanId = *GenTestSpanID()
		lr.Attributes = make([]KeyValue, len(attrKeys))
		for k := range attrKeys {
			lr.Attributes[k] = stringAttr(attrKeys[k], attrVals[k])
		}
		sl.LogRecords = append(sl.LogRecords, lr)
		if i%64 != 63 {
			continue
		}
		rl.ScopeLogs = []*ScopeLogs{sl}
		req.ResourceLogs = []*ResourceLogs{rl}
		if req.SizeProto() >= targetBytes {
			break
		}
	}

	buf := make([]byte, req.SizeProto())
	req.MarshalProto(buf)
	return buf
}

func TestGenTracePayloadSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10MB payload construction in short mode")
	}
	buf := bench10MBTracePayload()
	if len(buf) < benchPayloadBytes {
		t.Fatalf("payload size = %d, want at least %d", len(buf), benchPayloadBytes)
	}
}

func TestGenLogsPayloadSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10MB payload construction in short mode")
	}
	buf := bench10MBLogsPayload()
	if len(buf) < benchPayloadBytes {
		t.Fatalf("payload size = %d, want at least %d", len(buf), benchPayloadBytes)
	}
}

func BenchmarkUnmarshalProto10MB(b *testing.B) {
	benchmarkUnmarshalProto10MB(b, bench10MBTracePayload(), func() interface {
		UnmarshalProto([]byte) error
		UnmarshalProtoUnsafe([]byte) error
	} {
		return NewExportTraceServiceRequest()
	})
}

func BenchmarkUnmarshalProtoLogs10MB(b *testing.B) {
	benchmarkUnmarshalProto10MB(b, bench10MBLogsPayload(), func() interface {
		UnmarshalProto([]byte) error
		UnmarshalProtoUnsafe([]byte) error
	} {
		return NewExportLogsServiceRequest()
	})
}

func BenchmarkLogsUnsafeGRPCBuffer10MB(b *testing.B) {
	payload := bench10MBLogsPayload()
	single := mem.BufferSlice{mem.SliceBuffer(payload)}
	framed := splitBufferSlice(payload, 16<<10)

	b.Run("UnmarshalOnly", func(b *testing.B) {
		b.SetBytes(int64(len(payload)))
		b.ReportAllocs()
		for b.Loop() {
			if err := NewExportLogsServiceRequest().UnmarshalProtoUnsafe(payload); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SingleBufferNoCopy", func(b *testing.B) {
		b.SetBytes(int64(len(payload)))
		b.ReportAllocs()
		for b.Loop() {
			if err := NewExportLogsServiceRequest().UnmarshalProtoUnsafe(single[0].ReadOnlyData()); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MaterializeSingleThenUnmarshal", func(b *testing.B) {
		b.SetBytes(int64(len(payload)))
		b.ReportAllocs()
		for b.Loop() {
			raw := single.Materialize()
			if err := NewExportLogsServiceRequest().UnmarshalProtoUnsafe(raw); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Materialize16KiBFramesThenUnmarshal", func(b *testing.B) {
		b.SetBytes(int64(len(payload)))
		b.ReportAllocs()
		for b.Loop() {
			raw := framed.Materialize()
			if err := NewExportLogsServiceRequest().UnmarshalProtoUnsafe(raw); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func splitBufferSlice(payload []byte, frameSize int) mem.BufferSlice {
	var out mem.BufferSlice
	for len(payload) > 0 {
		n := frameSize
		if n > len(payload) {
			n = len(payload)
		}
		out = append(out, mem.SliceBuffer(payload[:n]))
		payload = payload[n:]
	}
	return out
}

func benchmarkUnmarshalProto10MB(b *testing.B, buf []byte, newDest func() interface {
	UnmarshalProto([]byte) error
	UnmarshalProtoUnsafe([]byte) error
}) {
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()

	b.Run("Safe", func(b *testing.B) {
		b.SetBytes(int64(len(buf)))
		b.ReportAllocs()
		for b.Loop() {
			if err := newDest().UnmarshalProto(buf); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Unsafe", func(b *testing.B) {
		b.SetBytes(int64(len(buf)))
		b.ReportAllocs()
		for b.Loop() {
			if err := newDest().UnmarshalProtoUnsafe(buf); err != nil {
				b.Fatal(err)
			}
		}
	})
}
