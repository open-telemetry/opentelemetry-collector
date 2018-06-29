// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"bytes"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/tinylib/msgp/msgp"
)

func TestPayload(t *testing.T) {
	t.Run("partitioning", func(t *testing.T) {
		p := newPayload()
		prevSize := 0
		for i, tt := range []struct {
			span         *ddSpan           // span to add
			traceLengths map[uint64]uint64 // maps traces to their expected length
		}{
			{
				span:         makeSpan(100),
				traceLengths: map[uint64]uint64{100: 1},
			},
			{
				span:         makeSpan(100),
				traceLengths: map[uint64]uint64{100: 2},
			},
			{
				span:         makeSpan(100),
				traceLengths: map[uint64]uint64{100: 3},
			},
			{
				span:         makeSpan(250),
				traceLengths: map[uint64]uint64{100: 3, 250: 1},
			},
			{
				span:         makeSpan(325),
				traceLengths: map[uint64]uint64{100: 3, 250: 1, 325: 1},
			},
			{
				span:         makeSpan(250),
				traceLengths: map[uint64]uint64{100: 3, 250: 2, 325: 1},
			},
		} {
			err := p.add(tt.span)
			if err != nil {
				t.Fatalf("%d: %v", i, err)
			}
			for id, total := range tt.traceLengths {
				if got := p.traces[id].count; got != total {
					t.Fatalf("%d: count mismatch at trace ID %d, expected %d, got %d", i, id, total, got)
				}
			}
			if p.size() <= prevSize {
				t.Fatalf("%d: expected a size above %d, got %d", i, prevSize, p.size())
			}
			prevSize = p.size()
		}
	})

	t.Run("size", func(t *testing.T) {
		p := newPayload()
		if p.size() != 0 {
			t.Fatal("wanted 0")
		}
		fillPayload(t, p)
		total := arrayHeaderSize(uint64(len(testPayload)))
		for _, tr := range p.traces {
			total += tr.size()
		}
		if total != p.size() {
			t.Fatal("size is off")
		}
	})

	t.Run("decode", func(t *testing.T) {
		p := newPayload()
		// run the test twice to test reset
		for i := 0; i < 1; i++ {
			p.reset()
			fillPayload(t, p)
			buf := p.buffer()
			var got ddPayload
			err := msgp.Decode(buf, &got)
			if err != nil {
				t.Fatal(err)
			}
			// use two loops to compare because the element order might differ
			for _, trc := range testPayload {
				var found bool
				for _, trc2 := range got {
					if reflect.DeepEqual(trc, trc2) {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("integrity error")
				}
			}
		}
	})
}

func TestPackedSpans(t *testing.T) {
	t.Run("integrity", func(t *testing.T) {
		// whatever we push into the packedSpans should allow us to read the same content
		// as would have been encoded by the encoder.
		ss := new(packedSpans)
		buf := new(bytes.Buffer)
		for _, n := range []int{10, 1 << 10, 1 << 17} {
			t.Run(strconv.Itoa(n), func(t *testing.T) {
				ss.reset()
				spanList := makeTrace(n)
				for _, span := range spanList {
					if err := ss.add(&span); err != nil {
						t.Fatal(err)
					}
				}
				buf.Reset()
				err := msgp.Encode(buf, spanList)
				if err != nil {
					t.Fatal(err)
				}
				if ss.count != uint64(n) {
					t.Fatalf("count mismatch: expected %d, got %d", ss.count, n)
				}
				got := ss.bytes()
				if len(got) == 0 {
					t.Fatal("0 bytes")
				}
				if !bytes.Equal(buf.Bytes(), got) {
					t.Fatalf("content mismatch")
				}
			})
		}
	})

	t.Run("size", func(t *testing.T) {
		ss := new(packedSpans)
		if ss.size() != 0 {
			t.Fatalf("expected 0, got %d", ss.size())
		}
		if err := ss.add(&ddSpan{SpanID: 1}); err != nil {
			t.Fatal(err)
		}
		if ss.size() <= 0 {
			t.Fatal("got 0")
		}
	})

	t.Run("decode", func(t *testing.T) {
		// ensure that whatever we push into the span slice can be decoded by the decoder.
		ss := new(packedSpans)
		for _, n := range []int{10, 1 << 10} {
			t.Run(strconv.Itoa(n), func(t *testing.T) {
				ss.reset()
				for i := 0; i < n; i++ {
					if err := ss.add(&ddSpan{SpanID: uint64(i)}); err != nil {
						t.Fatal(err)
					}
				}
				var got ddTrace
				err := msgp.Decode(bytes.NewReader(ss.bytes()), &got)
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	})
}

// makeTrace returns a ddTrace of size n.
func makeTrace(n int) ddTrace {
	ddt := make(ddTrace, n)
	for i := 0; i < n; i++ {
		span := ddSpan{SpanID: uint64(i)}
		ddt[i] = span
	}
	return ddt
}

// idSeed is the starting number from which the generated span IDs are incremented.
var idSeed uint64 = 123

// makeSpan returns a new span having id as the trace ID.
func makeSpan(id uint64) *ddSpan {
	atomic.AddUint64(&idSeed, 1)
	return &ddSpan{TraceID: id, SpanID: idSeed}
}

// testPayload returns a payload used for testing.
var testPayload = ddPayload{
	ddTrace{*makeSpan(1), *makeSpan(1), *makeSpan(1)},
	ddTrace{*makeSpan(2), *makeSpan(2)},
	ddTrace{*makeSpan(3), *makeSpan(3), *makeSpan(3), *makeSpan(3)},
}

// fillPayload adds the traces from testPayload to payload.
func fillPayload(t *testing.T, p *payload) {
	for _, spans := range testPayload {
		for _, span := range spans {
			if err := p.add(&span); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func BenchmarkThroughput(b *testing.B) {
	p := newPayload()
	b.SetBytes(int64(flushThreshold))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.reset()
		for p.size() < flushThreshold {
			if err := p.add(spanPairs["tags"].dd); err != nil {
				b.Fatal(err)
			}
		}
	}
}
