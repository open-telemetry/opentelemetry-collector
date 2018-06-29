package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestSpanJSON(t *testing.T) {
	var (
		span1    SpanModel
		span2    SpanModel
		parentID = ID(1003)
		sampled  = true
		tags     = make(map[string]string)
	)
	tags["myKey"] = "myValue"
	tags["another"] = "tag"

	span1 = SpanModel{
		SpanContext: SpanContext{
			TraceID: TraceID{
				High: 1001,
				Low:  1002,
			},
			ID:       ID(1004),
			ParentID: &parentID,
			Debug:    true,
			Sampled:  &sampled,
			Err:      errors.New("dummy"),
		},
		Name:      "myMethod",
		Kind:      Server,
		Timestamp: time.Now().Add(-100 * time.Millisecond),
		Duration:  50 * time.Millisecond,
		Shared:    true,
		LocalEndpoint: &Endpoint{
			ServiceName: "myService",
			IPv4:        net.IPv4(127, 0, 0, 1),
			IPv6:        net.IPv6loopback,
		},
		RemoteEndpoint: nil,
		Annotations: []Annotation{
			{time.Now().Add(-90 * time.Millisecond), "myAnnotation"},
		},
		Tags: tags,
	}

	b, err := json.Marshal(&span1)
	if err != nil {
		t.Errorf("expected successful serialization to JSON, got error: %+v", err)
	}

	err = json.Unmarshal(b, &span2)
	if err != nil {
		t.Errorf("expected successful deserialization from JSON, got error: %+v", err)
	}

	/* remove items from span1 which should not have exported */
	span1.Sampled = nil
	span1.Err = nil

	// trim resolution back to microseconds (Zipkin's smallest time unit)
	span1.Timestamp = span1.Timestamp.Round(time.Microsecond)
	for idx := range span1.Annotations {
		span1.Annotations[idx].Timestamp = span1.Annotations[idx].Timestamp.Round(time.Microsecond)
	}

	if !reflect.DeepEqual(span1, span2) {
		t.Errorf("want SpanModel: %+v, have: %+v", span1, span2)
	}
}

func TestEmptyTraceID(t *testing.T) {
	var (
		span SpanModel
		b    = []byte(`{"traceId":"","id":"1"}`)
	)

	if err := json.Unmarshal(b, &span); err == nil {
		t.Errorf("Unmarshal should have failed with error, have: %+v", span)
	}
}

func TestEmptySpanID(t *testing.T) {
	var (
		span SpanModel
		b    = []byte(`{"traceId":"1","id":""}`)
	)

	if err := json.Unmarshal(b, &span); err == nil {
		t.Errorf("Unmarshal should have failed with error, have: %+v", span)
	}
}

func TestSpanEmptyTimeStamp(t *testing.T) {
	var (
		span1 SpanModel
		span2 SpanModel
		ts    time.Time
	)

	span1 = SpanModel{
		SpanContext: SpanContext{
			TraceID: TraceID{
				Low: 1,
			},
			ID: 1,
		},
	}

	b, err := json.Marshal(span1)
	if err != nil {
		t.Fatalf("unable to marshal span: %+v", err)
	}

	if err := json.Unmarshal(b, &span2); err != nil {
		t.Fatalf("unable to unmarshal span: %+v", err)
	}

	if want, have := ts, span2.Timestamp; want != have {
		t.Errorf("Timestamp want %s, have %s", want, have)
	}
}

func TestSpanDurationRounding(t *testing.T) {
	durations := []struct {
		nano  time.Duration
		micro time.Duration
	}{
		{0, 0},
		{1, 1000},
		{999, 1000},
		{1000, 1000},
		{1001, 1000},
		{1499, 1000},
		{1500, 2000},
		{2000, 2000},
		{2001, 2000},
		{2499, 2000},
		{2500, 3000},
		{2999, 3000},
		{3000, 3000},
	}

	for i, duration := range durations {
		span := SpanModel{
			SpanContext: SpanContext{
				TraceID: TraceID{Low: 1},
				ID:      ID(1),
			},
			Timestamp: time.Now(),
			Duration:  duration.nano,
		}

		b, err := json.Marshal(span)
		if err != nil {
			t.Fatalf("span marshal failed: %+v", err)
		}

		span2 := SpanModel{}

		if err := json.Unmarshal(b, &span2); err != nil {
			t.Fatalf("span unmarshal failed: %+v", err)
		}

		if want, have := duration.micro, span2.Duration; want != have {
			t.Errorf("[%d] Duration want %d, have %d", i, want, have)
		}
	}
}

func TestSpanNegativeDuration(t *testing.T) {
	var (
		err  error
		span SpanModel
		b    = []byte(`{"duration":-1}`)
	)

	if err = json.Unmarshal(b, &span); err == nil {
		t.Errorf("Unmarshal should have failed with error, have: %+v", span)
	}

	span = SpanModel{
		SpanContext: SpanContext{
			TraceID: TraceID{Low: 1},
			ID:      ID(1),
		},
		Timestamp: time.Now(),
		Duration:  -1 * time.Nanosecond,
	}

	if _, err = json.Marshal(span); err == nil {
		t.Fatalf("Span Marshal Error expected, have nil")
	}

	want := fmt.Sprintf(
		"json: error calling MarshalJSON for type model.SpanModel: %s",
		ErrValidDurationRequired.Error(),
	)

	if have := err.Error(); want != have {
		t.Errorf("Span Marshal Error want %s, have %s", want, have)
	}
}

func TestSpanNegativeTimestamp(t *testing.T) {
	var (
		err  error
		span SpanModel
		b    = []byte(`{"timestamp":-1}`)
	)

	if err = json.Unmarshal(b, &span); err == nil {
		t.Errorf("Unmarshal should have failed with error, have: %+v", span)
	}

	span = SpanModel{
		SpanContext: SpanContext{
			TraceID: TraceID{Low: 1},
			ID:      ID(1),
		},
		Timestamp: time.Unix(-1, 0),
	}

	if _, err = json.Marshal(span); err == nil {
		t.Fatalf("Span Marshal Error expected, have nil")
	}

	want := fmt.Sprintf(
		"json: error calling MarshalJSON for type model.SpanModel: %s",
		ErrValidTimestampRequired.Error(),
	)
	if have := err.Error(); want != have {
		t.Errorf("Span Marshal Error want %s, have %s", want, have)
	}

}
