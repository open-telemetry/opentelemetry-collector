package zipkin

import (
	"reflect"
	"testing"
	"time"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

func TestNoopContext(t *testing.T) {
	var (
		span     Span
		sc       model.SpanContext
		parentID = model.ID(3)
		tr, _    = NewTracer(
			reporter.NewNoopReporter(),
			WithNoopSpan(true),
			WithSampler(neverSample),
			WithSharedSpans(true),
		)
	)

	sc = model.SpanContext{
		TraceID:  model.TraceID{High: 1, Low: 2},
		ID:       model.ID(4),
		ParentID: &parentID,
		Debug:    false,     // debug must be false
		Sampled:  new(bool), // bool must be pointer to false
	}

	span = tr.StartSpan("testNoop", Parent(sc), Kind(model.Server))

	noop, ok := span.(*noopSpan)
	if !ok {
		t.Fatalf("Span type want %s, have %s", reflect.TypeOf(&spanImpl{}), reflect.TypeOf(span))
	}

	if have := noop.Context(); !reflect.DeepEqual(sc, have) {
		t.Errorf("Context want %+v, have %+v", sc, have)
	}

	span.Tag("dummy", "dummy")
	span.Annotate(time.Now(), "dummy")
	span.SetName("dummy")
	span.SetRemoteEndpoint(nil)
	span.Flush()
}
