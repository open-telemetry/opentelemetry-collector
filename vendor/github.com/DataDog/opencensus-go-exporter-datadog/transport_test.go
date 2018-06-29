// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
)

func TestTransport(t *testing.T) {
	_, ok := os.LookupEnv("INTEGRATION")
	if !ok {
		t.Skip("to run: set the INTEGRATION environment variable and have the agent running")
	}
	p := newPayload()
	for _, span := range []*ddSpan{
		testSpan(1234, "abc", "qwe"),
		testSpan(1234, "abc1", "qwe"),
		testSpan(4567, "abc2", "qwe"),
		testSpan(4567, "abc2", "qwe"),
	} {
		p.add(span)
	}
	trans := newTransport("")
	err := trans.upload(p.buffer(), len(p.traces))
	if err != nil {
		t.Fatal(err)
	}
}

// testSpan returns a minimally valid span that the agent will accept
// through its normalization process.
func testSpan(traceID uint64, name, service string) *ddSpan {
	now := time.Now()
	start := now.UnixNano()
	duration := now.Add(time.Minute).UnixNano() - start
	return &ddSpan{
		TraceID:  traceID,
		SpanID:   atomic.AddUint64(&idSeed, 1),
		Name:     name,
		Resource: name,
		Service:  service,
		Start:    start,
		Duration: duration,
		Metrics:  map[string]float64{samplingPriorityKey: ext.PriorityAutoKeep},
	}
}
