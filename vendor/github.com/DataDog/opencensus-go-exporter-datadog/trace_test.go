// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"bytes"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/tinylib/msgp/msgp"
)

const (
	// testFlushInterval is the flush interval that will be used for the
	// duration of the tests.
	testFlushInterval = 24 * time.Hour

	// testFlushThreshold is the flush threshold that will be used for the
	// duration of the tests.
	testFlushThreshold = 1e3

	// testInChannelSize is the input channel's buffer size that will be used
	// for the duration of the tests.
	testInChannelSize = 1000
)

func TestMain(m *testing.M) {
	o1, o2, o3 := flushInterval, flushThreshold, inChannelSize
	flushInterval = testFlushInterval
	flushThreshold = testFlushThreshold
	inChannelSize = testInChannelSize

	defer func() {
		flushInterval, flushThreshold, inChannelSize = o1, o2, o3
	}()

	os.Exit(m.Run())
}

func TestTraceExporter(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Run("service", func(t *testing.T) {
		me := newTraceExporter(Options{})
		defer me.stop()
		if me.opts.Service == "" {
			t.Fatal("service should never be empty")
		}
	})

	t.Run("threshold", func(t *testing.T) {
		me := newTestTraceExporter(t)
		defer me.stop()
		span := spanPairs["tags"].oc
		count := 5 // 5 spans should take us overboard
		for i := 0; i < count; i++ {
			me.exportSpan(span)
		}
		time.Sleep(time.Millisecond) // wait for recv
		me.wg.Wait()                 // wait for flush
		flushed := me.payloads()
		eq := equalFunc(t)
		eq(len(flushed), 1)
		eq(len(flushed[0][0]), count)
	})

	t.Run("stop", func(t *testing.T) {
		me := newTestTraceExporter(t)
		me.exportSpan(spanPairs["root"].oc)

		time.Sleep(time.Millisecond) // wait for recv

		me.stop()
		if len(me.payloads()) != 1 {
			t.Fatalf("expected to flush 1, got %d", len(me.payloads()))
		}
	})
}

// testTraceExporter wraps a traceExporter, recording all flushed payloads.
type testTraceExporter struct {
	*traceExporter
	t *testing.T

	mu      sync.RWMutex
	flushed []ddPayload
}

func newTestTraceExporter(t *testing.T) *testTraceExporter {
	te := newTraceExporter(Options{Service: "mock.exporter"})
	me := &testTraceExporter{traceExporter: te, flushed: make([]ddPayload, 0)}
	me.traceExporter.uploadFn = me.uploadFn
	return me
}

// payloads returns all payloads that were uploaded by this exporter.
func (me *testTraceExporter) payloads() []ddPayload {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return me.flushed
}

func (me *testTraceExporter) uploadFn(buf *bytes.Buffer, _ int) error {
	var ddp ddPayload
	if err := msgp.Decode(buf, &ddp); err != nil {
		me.t.Fatal(err)
	}
	me.mu.Lock()
	me.flushed = append(me.flushed, ddp)
	me.mu.Unlock()
	return nil
}
