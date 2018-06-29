package zipkin

import (
	"reflect"
	"testing"
	"time"

	"github.com/openzipkin/zipkin-go/reporter"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

func TestSpanNameUpdate(t *testing.T) {
	var (
		oldName = "oldName"
		newName = "newName"
	)

	tracer, _ := NewTracer(reporter.NewNoopReporter())

	span := tracer.StartSpan(oldName)

	if want, have := oldName, span.(*spanImpl).Name; want != have {
		t.Errorf("Name want %q, have %q", want, have)
	}

	span.SetName(newName)

	if want, have := newName, span.(*spanImpl).Name; want != have {
		t.Errorf("Name want %q, have %q", want, have)
	}
}

func TestRemoteEndpoint(t *testing.T) {
	tracer, err := NewTracer(reporter.NewNoopReporter())
	if err != nil {
		t.Fatalf("expected valid tracer, got error: %+v", err)
	}

	ep1, err := NewEndpoint("myService", "www.google.com:80")

	if err != nil {
		t.Fatalf("expected valid endpoint, got error: %+v", err)
	}

	span := tracer.StartSpan("test", RemoteEndpoint(ep1))

	if !reflect.DeepEqual(span.(*spanImpl).RemoteEndpoint, ep1) {
		t.Errorf("RemoteEndpoint want %+v, have %+v", ep1, span.(*spanImpl).RemoteEndpoint)
	}

	ep2, err := NewEndpoint("otherService", "www.microsoft.com:443")

	if err != nil {
		t.Fatalf("expected valid endpoint, got error: %+v", err)
	}

	span.SetRemoteEndpoint(ep2)

	if !reflect.DeepEqual(span.(*spanImpl).RemoteEndpoint, ep2) {
		t.Errorf("RemoteEndpoint want %+v, have %+v", ep1, span.(*spanImpl).RemoteEndpoint)
	}

	span.SetRemoteEndpoint(nil)

	if have := span.(*spanImpl).RemoteEndpoint; have != nil {
		t.Errorf("RemoteEndpoint want nil, have %+v", have)
	}
}

func TestTagsSpanOption(t *testing.T) {
	tracerTags := map[string]string{
		"key1": "value1",
		"key2": "will_be_overwritten",
	}
	tracer, err := NewTracer(reporter.NewNoopReporter(), WithTags(tracerTags))
	if err != nil {
		t.Fatalf("expected valid tracer, got error: %+v", err)
	}

	spanTags := map[string]string{
		"key2": "value2",
		"key3": "value3",
	}
	span := tracer.StartSpan("test", Tags(spanTags))
	defer span.Finish()

	allTags := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	if want, have := allTags, span.(*spanImpl).Tags; !reflect.DeepEqual(want, have) {
		t.Errorf("Tags want: %+v, have: %+v", want, have)
	}
}

func TestFlushOnFinishSpanOption(t *testing.T) {
	rec := recorder.NewReporter()
	defer rec.Close()

	tracer, _ := NewTracer(rec)

	span := tracer.StartSpan("test")
	time.Sleep(5 * time.Millisecond)
	span.Finish()

	spans := rec.Flush()

	if want, have := 1, len(spans); want != have {
		t.Errorf("Spans want: %d, have %d", want, have)
	}

	span = tracer.StartSpan("test", FlushOnFinish(false))
	time.Sleep(5 * time.Millisecond)
	span.Finish()

	spans = rec.Flush()

	if want, have := 0, len(spans); want != have {
		t.Errorf("Spans want: %d, have %d", want, have)
	}

	span.Tag("post", "finish")
	span.Flush()

	spans = rec.Flush()

	if want, have := 1, len(spans); want != have {
		t.Errorf("Spans want: %d, have %d", want, have)
	}

	if want, have := map[string]string{"post": "finish"}, spans[0].Tags; !reflect.DeepEqual(want, have) {
		t.Errorf("Tags want: %+v, have: %+v", want, have)
	}

}
