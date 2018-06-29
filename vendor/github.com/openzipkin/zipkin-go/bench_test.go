package zipkin_test

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/idgenerator"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"google.golang.org/grpc/metadata"
)

const (
	b3HTTP = "b3-http"
	b3GRPC = "b3-grpc"
)

var tags []string

func init() {
	var (
		traceID model.TraceID
		gen     = idgenerator.NewRandom64()
	)

	tags = make([]string, 1000)
	for j := 0; j < len(tags); j++ {
		tags[j] = fmt.Sprintf("%d", gen.SpanID(traceID))
	}

}

func addAnnotationsAndTags(sp zipkin.Span, numAnnotation, numTag int) {
	for j := 0; j < numAnnotation; j++ {
		sp.Annotate(time.Now(), "event")
	}

	for j := 0; j < numTag; j++ {
		sp.Tag(tags[j], "")
	}
}

func benchmarkWithOps(b *testing.B, numAnnotation, numTag int) {
	var (
		r    countingRecorder
		t, _ = zipkin.NewTracer(&r)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sp := t.StartSpan("test")
		addAnnotationsAndTags(sp, numAnnotation, numTag)
		sp.Finish()
	}

	b.StopTimer()

	if int(r) != b.N {
		b.Fatalf("missing traces: want %d, have %d", b.N, r)
	}
}

func BenchmarkSpan_Empty(b *testing.B) {
	benchmarkWithOps(b, 0, 0)
}

func BenchmarkSpan_100Annotations(b *testing.B) {
	benchmarkWithOps(b, 100, 0)
}

func BenchmarkSpan_1000Annotations(b *testing.B) {
	benchmarkWithOps(b, 1000, 0)
}

func BenchmarkSpan_100Tags(b *testing.B) {
	benchmarkWithOps(b, 0, 100)
}

func BenchmarkSpan_1000Tags(b *testing.B) {
	benchmarkWithOps(b, 0, 1000)
}

func benchmarkInject(b *testing.B, propagationType string) {
	var (
		r         countingRecorder
		injector  propagation.Injector
		tracer, _ = zipkin.NewTracer(&r)
	)

	switch propagationType {
	case b3HTTP:
		req, _ := http.NewRequest("GET", "/", nil)
		injector = b3.InjectHTTP(req)
	case b3GRPC:
		md := metadata.MD{}
		injector = b3.InjectGRPC(&md)
	default:
		b.Fatalf("unknown injector: %s", propagationType)
	}

	sp := tracer.StartSpan("testing")
	addAnnotationsAndTags(sp, 0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := injector(sp.Context()); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkExtract(b *testing.B, propagationType string) {
	var (
		r         countingRecorder
		tracer, _ = zipkin.NewTracer(&r)
	)

	sp := tracer.StartSpan("testing")

	switch propagationType {
	case b3HTTP:
		req, _ := http.NewRequest("GET", "/", nil)
		b3.InjectHTTP(req)(sp.Context())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = b3.ExtractHTTP(copyRequest(req))
		}

	case b3GRPC:
		md := metadata.MD{}
		b3.InjectGRPC(&md)(sp.Context())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			md2 := md.Copy()
			if _, err := b3.ExtractGRPC(&md2)(); err != nil {
				b.Fatal(err)
			}
		}
	default:
		b.Fatalf("unknown propagation type: %s", propagationType)
	}
}

func BenchmarkInject_B3_HTTP_Empty(b *testing.B) {
	benchmarkInject(b, b3HTTP)
}

func BenchmarkInject_B3_GRPC_Empty(b *testing.B) {
	benchmarkInject(b, b3GRPC)
}

func BenchmarkExtract_B3_HTTP_Empty(b *testing.B) {
	benchmarkExtract(b, b3HTTP)
}

func BenchmarkExtract_B3_GRPC_Empty(b *testing.B) {
	benchmarkExtract(b, b3GRPC)
}

type countingRecorder int32

func (c *countingRecorder) Send(_ model.SpanModel) {
	atomic.AddInt32((*int32)(c), 1)
}

func (c *countingRecorder) Close() error { return nil }

func copyRequest(req *http.Request) *http.Request {
	r, _ := http.NewRequest("GET", "/", nil)
	for k, v := range req.Header {
		r.Header[k] = v
	}
	return r
}
