// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestGatePassThrough(t *testing.T) {
	sink := new(consumertest.TracesSink)
	g := NewTraces(sink)

	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()

	err := g.ConsumeTraces(context.Background(), td)
	require.NoError(t, err)
	assert.Len(t, sink.AllTraces(), 1)
}

func TestGateCapabilities(t *testing.T) {
	sink := new(consumertest.TracesSink)
	g := NewTraces(sink)
	assert.Equal(t, sink.Capabilities(), g.Capabilities())
}

func TestGatePauseBlocksCallers(t *testing.T) {
	sink := new(consumertest.TracesSink)
	g := NewTraces(sink)

	old := g.Pause()
	assert.Equal(t, sink, old)

	blocked := make(chan struct{})
	go func() {
		td := ptrace.NewTraces()
		_ = g.ConsumeTraces(context.Background(), td)
		close(blocked)
	}()

	time.Sleep(50 * time.Millisecond)
	select {
	case <-blocked:
		t.Fatal("caller should be blocked during pause")
	default:
	}

	newSink := new(consumertest.TracesSink)
	g.Resume(newSink)

	select {
	case <-blocked:
	case <-time.After(time.Second):
		t.Fatal("caller should have unblocked after resume")
	}

	assert.Len(t, newSink.AllTraces(), 1)
	assert.Len(t, sink.AllTraces(), 0)
}

func TestGatePauseDrainsInflight(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	blockingConsumer := &blockingTracesConsumer{
		release: release,
		entered: entered,
	}
	g := NewTraces(blockingConsumer)

	inflightDone := make(chan struct{})
	go func() {
		td := ptrace.NewTraces()
		_ = g.ConsumeTraces(context.Background(), td)
		close(inflightDone)
	}()

	<-entered

	pauseDone := make(chan struct{})
	go func() {
		g.Pause()
		close(pauseDone)
	}()

	time.Sleep(50 * time.Millisecond)
	select {
	case <-pauseDone:
		t.Fatal("pause should block until in-flight calls drain")
	default:
	}

	close(release)
	<-inflightDone

	select {
	case <-pauseDone:
	case <-time.After(time.Second):
		t.Fatal("pause should complete after in-flight calls drain")
	}
}

func TestGateConcurrentPauseResume(t *testing.T) {
	sink1 := new(consumertest.TracesSink)
	g := NewTraces(sink1)

	const numGoroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	var totalSent atomic.Int64

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				td := ptrace.NewTraces()
				td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				if err := g.ConsumeTraces(context.Background(), td); err == nil {
					totalSent.Add(1)
				}
			}
		}()
	}

	for i := 0; i < 5; i++ {
		time.Sleep(10 * time.Millisecond)
		old := g.Pause()
		_ = old
		newSink := new(consumertest.TracesSink)
		g.Resume(newSink)
	}

	wg.Wait()
	assert.Equal(t, int64(numGoroutines*iterations), totalSent.Load())
}

// --- Benchmarks ---

func BenchmarkGatePassThrough(b *testing.B) {
	sink := &noopTracesConsumer{}
	g := NewTraces(sink)
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_ = g.ConsumeTraces(ctx, td)
	}
}

func BenchmarkGateEnterLeave(b *testing.B) {
	sink := &noopTracesConsumer{}
	g := New[consumer.Traces](sink)

	b.ResetTimer()
	for b.Loop() {
		gen := g.Enter()
		g.Leave(gen)
	}
}

func BenchmarkGateParallel(b *testing.B) {
	sink := &noopTracesConsumer{}
	g := NewTraces(sink)
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	ctx := context.Background()

	b.SetParallelism(runtime.GOMAXPROCS(0))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = g.ConsumeTraces(ctx, td)
		}
	})
}

func BenchmarkGatePauseResume(b *testing.B) {
	sink := &noopTracesConsumer{}
	g := New[consumer.Traces](sink)

	b.ResetTimer()
	for b.Loop() {
		old := g.Pause()
		g.Resume(old)
	}
}

func BenchmarkGateMultipleGates5(b *testing.B) {
	gates := make([]*Gate[consumer.Traces], 5)
	for i := 0; i < 5; i++ {
		gates[i] = New[consumer.Traces](&noopTracesConsumer{})
	}

	b.ResetTimer()
	for b.Loop() {
		for _, g := range gates {
			gen := g.Enter()
			g.Leave(gen)
		}
	}
}

// --- helpers ---

type noopTracesConsumer struct{}

func (n *noopTracesConsumer) ConsumeTraces(context.Context, ptrace.Traces) error {
	return nil
}

func (n *noopTracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

type blockingTracesConsumer struct {
	release chan struct{}
	entered chan struct{}
}

func (b *blockingTracesConsumer) ConsumeTraces(_ context.Context, _ ptrace.Traces) error {
	close(b.entered)
	<-b.release
	return nil
}

func (b *blockingTracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}
