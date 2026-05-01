// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package gate provides an atomic consumer gate that can be inserted between
// pipeline components. It allows the underlying consumer to be replaced at
// runtime using a Pause/Resume protocol that safely drains in-flight calls
// before the swap.
//
// # Overview
//
// The gate uses a "generation" pattern where each consumer installation creates
// a new generation object. The core invariant is:
//
//   - Data-flow goroutines acquire a generation (incrementing its in-flight counter),
//     do their work, then release it (decrementing the counter).
//   - A swap operation installs a new generation and waits for the old generation's
//     in-flight counter to reach zero before returning.
//
// This is implemented with atomics for the fast path and channels only for the
// blocking swap path.
//
// # Swap Path: Pause / Resume (component replacement)
//
// Component replacement uses a two-phase Pause/Resume protocol:
//
//	old := gate.Pause()     // (1) block new callers, (2) drain in-flight
//	old.Shutdown(ctx)       // safe: no one is using the old consumer
//	new := factory.Create() // safe: old released resources (ports, files, etc.)
//	gate.Resume(new)        // unblock callers, data flows through new consumer
//
// Pause works by:
//
//  1. Creating a "blocker" generation with blocked=true and an open ready channel.
//  2. Atomically swapping it in: gate.gen.Swap(blocker)
//  3. Applying a "drain bias" to the old generation's inflight counter:
//     oldGen.inflight.Add(-drainBias)
//     This makes the counter deeply negative so we can distinguish
//     "zero real in-flight calls" from "some calls still active".
//     Real in-flight count = counter + drainBias.
//  4. If real count > 0, waiting on <-oldGen.done for the last caller to signal.
//
// Resume works by:
//
//  1. Setting the blocker's consumer to the new consumer.
//  2. Setting blocked=false (future Enter() calls skip the channel wait).
//  3. Closing the ready channel (unblocking all callers waiting in Enter()).
//
// # Drain Bias Mechanism
//
// The drain bias is a large constant (1<<30) subtracted from the inflight counter
// when a generation starts draining. This avoids needing a separate "draining" flag
// and race-free check. The inflight counter values mean:
//
//	>= 0           : normal operation, value = number of in-flight calls
//	== -drainBias  : drained (bias subtracted, zero in-flight)
//	< 0, > -bias   : impossible during normal operation
//	-bias + N      : draining, N calls still in-flight
//
// Only the caller whose Add(-1) produces exactly -drainBias closes the done channel.
// Multiple concurrent decrements (including stale readers doing add/subtract pairs)
// cannot produce a false drain signal because their operations net to zero.
//
// The bias is subtracted (driving the counter negative) rather than added (driving
// it to a large positive) because during normal operation inflight is always >= 0.
// A negative counter is an unambiguous signal that the generation is draining —
// it cannot be confused with a legitimate in-flight count. Adding a large positive
// bias would produce a value that is technically indistinguishable from "many
// concurrent callers" and loses the invariant that negative = draining.
//
// # Race Scenarios and Correctness
//
// Scenario A — Reader fully commits before Pause:
//
//	Reader: Load→gen, Add(1)→1, doubleCheck=gen ✓ → committed
//	Pauser: Swap(blocker), inflight.Add(-bias)→1-bias → waits on done
//	Reader: ConsumeTraces done, Leave → Add(-1)→-bias → close(done)
//	Pauser: unblocked, returns old consumer safely ✓
//
// Scenario B — Pause wins the race, reader detects stale generation:
//
//	Reader: Load→oldGen                              (gets old pointer)
//	Pauser: Swap(blocker), Add(-bias)→-bias → no wait, returns immediately
//	Reader: blocked.Load()→false (checking OLD gen)
//	Reader: Add(1)→-bias+1
//	Reader: doubleCheck: Load()→blocker ≠ oldGen → STALE
//	Reader: Add(-1)→-bias → close(done) (harmless, no one waiting)
//	Reader: retries → loads blocker → blocked=true → <-ready (blocks)
//	        ... Resume called ...
//	Reader: unblocked, retries → proceeds with new consumer ✓
//
//	KEY: The reader NEVER touches oldGen.consumer. It detects staleness
//	and retries. The old consumer is never called after Pause returns.
//
// Scenario C — Reader between blocked check and Add(1):
//
//	Reader: Load→oldGen, blocked=false
//	Pauser: Swap(blocker)                            (between reader's steps)
//	Reader: Add(1)→1 (on oldGen which is being drained)
//	Pauser: Add(-bias)→1-bias → waits on done
//	Reader: doubleCheck: Load()→blocker ≠ oldGen → STALE
//	Reader: Add(-1)→-bias → close(done)
//	Pauser: unblocked ✓
//	Reader: retries → sees blocker → blocks ✓
//
// Scenario D — Multiple blocked readers, then Resume:
//
//	Reader A & B: both blocked on <-blocker.ready
//	Resume: sets consumer, blocked=false, close(ready)
//	Reader A: unblocks, Load→blocker, blocked=false, Add(1), doubleCheck ✓
//	Reader B: unblocks, same path ✓
package gate // import "go.opentelemetry.io/collector/service/internal/gate"

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// drainBias is subtracted from the inflight counter to signal that a generation
// is draining. The value must be larger than any realistic number of concurrent
// in-flight calls.
const drainBias int64 = 1 << 30

// Consumer is the constraint for types that can be used with a Gate.
// All OpenTelemetry consumer interfaces (Traces, Metrics, Logs, Profiles)
// satisfy this constraint.
type Consumer interface {
	Capabilities() consumer.Capabilities
}

// generation holds a consumer and the synchronization state for one
// "era" of the gate. Each Pause/Resume cycle creates a new generation.
type generation[C Consumer] struct {
	consumer C
	inflight atomic.Int64
	done     chan struct{} // closed by last in-flight call when draining completes
	blocked  atomic.Bool   // true while paused — callers wait on ready
	ready    chan struct{} // closed by Resume to unblock waiting callers
}

// Gate is a concurrency-safe wrapper around a consumer that allows atomic
// replacement with pause/drain/resume semantics. The zero value is not
// usable; create one with New.
type Gate[C Consumer] struct {
	gen atomic.Pointer[generation[C]]
}

// New creates a Gate with an initial consumer.
func New[C Consumer](initial C) *Gate[C] {
	g := &Gate[C]{}
	gen := &generation[C]{
		consumer: initial,
		done:     make(chan struct{}),
		ready:    make(chan struct{}),
	}
	close(gen.ready) // immediately ready
	g.gen.Store(gen)
	return g
}

// Enter acquires the gate for a consumer call. Returns the current
// generation, which MUST be passed to Leave when the call is done.
//
// If the gate is paused, Enter blocks until Resume is called.
func (g *Gate[C]) Enter() *generation[C] {
	for {
		gen := g.gen.Load()

		// Fast path: blocked is false during normal operation.
		if gen.blocked.Load() {
			<-gen.ready // block until Resume
			continue
		}

		gen.inflight.Add(1)

		// Double-check: if the generation was swapped between
		// our first Load and the Add, undo and retry.
		if g.gen.Load() == gen {
			return gen
		}

		// Stale generation — undo the increment.
		if gen.inflight.Add(-1) == -drainBias {
			close(gen.done)
		}
	}
}

// Leave releases a generation acquired by Enter.
func (g *Gate[C]) Leave(gen *generation[C]) {
	if gen.inflight.Add(-1) == -drainBias {
		close(gen.done) // last one out signals drain complete
	}
}

// Pause blocks all new callers, drains in-flight calls, and returns
// the old consumer. After Pause returns:
//   - No calls are in-flight on the old consumer.
//   - All new callers are blocked until Resume is called.
//   - The returned consumer is safe to Shutdown.
//
// Must be followed by exactly one Resume call.
func (g *Gate[C]) Pause() C {
	blocker := &generation[C]{
		done:  make(chan struct{}),
		ready: make(chan struct{}),
	}
	blocker.blocked.Store(true)

	oldGen := g.gen.Swap(blocker)

	remaining := oldGen.inflight.Add(-drainBias) + drainBias
	if remaining > 0 {
		<-oldGen.done
	}

	return oldGen.consumer
}

// Resume installs a new consumer and unblocks all waiting callers.
// Must be called after Pause.
func (g *Gate[C]) Resume(newConsumer C) {
	gen := g.gen.Load()
	gen.consumer = newConsumer
	gen.blocked.Store(false)
	close(gen.ready)
}

// Capabilities returns the current consumer's capabilities.
// During a pause, callers are blocked in Enter so this always
// reflects the active consumer.
func (g *Gate[C]) Capabilities() consumer.Capabilities {
	return g.gen.Load().consumer.Capabilities()
}

// GetConsumer returns the generation's consumer.
func (gen *generation[C]) GetConsumer() C {
	return gen.consumer
}

// --- Signal-specific gate types ---
// These wrap Gate[C] and implement the corresponding consumer interface
// so they can be used as drop-in replacements in the pipeline graph.

// TracesGate wraps Gate[consumer.Traces] and implements consumer.Traces.
type TracesGate struct {
	*Gate[consumer.Traces]
}

// NewTraces creates a TracesGate with an initial consumer.
func NewTraces(initial consumer.Traces) *TracesGate {
	return &TracesGate{Gate: New(initial)}
}

func (g *TracesGate) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	gen := g.Enter()
	defer g.Leave(gen)
	return gen.consumer.ConsumeTraces(ctx, td)
}

// MetricsGate wraps Gate[consumer.Metrics] and implements consumer.Metrics.
type MetricsGate struct {
	*Gate[consumer.Metrics]
}

// NewMetrics creates a MetricsGate with an initial consumer.
func NewMetrics(initial consumer.Metrics) *MetricsGate {
	return &MetricsGate{Gate: New(initial)}
}

func (g *MetricsGate) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	gen := g.Enter()
	defer g.Leave(gen)
	return gen.consumer.ConsumeMetrics(ctx, md)
}

// LogsGate wraps Gate[consumer.Logs] and implements consumer.Logs.
type LogsGate struct {
	*Gate[consumer.Logs]
}

// NewLogs creates a LogsGate with an initial consumer.
func NewLogs(initial consumer.Logs) *LogsGate {
	return &LogsGate{Gate: New(initial)}
}

func (g *LogsGate) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	gen := g.Enter()
	defer g.Leave(gen)
	return gen.consumer.ConsumeLogs(ctx, ld)
}

// ProfilesGate wraps Gate[xconsumer.Profiles] and implements xconsumer.Profiles.
type ProfilesGate struct {
	*Gate[xconsumer.Profiles]
}

// NewProfiles creates a ProfilesGate with an initial consumer.
func NewProfiles(initial xconsumer.Profiles) *ProfilesGate {
	return &ProfilesGate{Gate: New(initial)}
}

func (g *ProfilesGate) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	gen := g.Enter()
	defer g.Leave(gen)
	return gen.consumer.ConsumeProfiles(ctx, pd)
}
