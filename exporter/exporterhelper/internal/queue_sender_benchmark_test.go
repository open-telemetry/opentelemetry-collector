// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

// BenchmarkQueueSender_Comparison is the evidence table promised in
// https://github.com/open-telemetry/opentelemetry-collector/issues/14080.
//
// Three concurrency strategies are compared:
//
//   StaticLow  – num_consumers=2 (conservative; safe under degradation, underutilises healthy capacity)
//   StaticHigh – num_consumers=50 (maximises throughput when healthy; amplifies overload under degradation)
//   ARC        – num_consumers=50 + AIMD token-pool middleware
//                (starts at limit=20, adapts between 5 and 50 based on RTT and error signals)
//
// Six backend scenarios:
//
//   Healthy         – 0 ms, 0 % errors (healthy baseline; ARC should ≈ StaticHigh)
//   BackendOverload – backend rejects requests above 10 concurrent in-flight (KEY scenario;
//                     ARC adapts to the limit; StaticHigh hammers past it and drops heavily)
//   LatencySpike    – 0 ms first half, 100 ms second half (soft signal; ARC reduces in-flight)
//   GradualLatency  – linear 0 → 100 ms ramp (ARC detects before errors appear)
//   ErrorSpike      – 25 % random retryable errors (ARC backs off gently; fewer total drops)
//   Recovery        – 100 ms first half then 0 ms (ARC recovers quickly; StaticLow does not)
//
// Expected outcomes:
//
//   ARC wins decisively:  BackendOverload (error_rate, dropped_count), Recovery (throughput)
//   ARC wins marginally:  Healthy (≈ StaticHigh), ErrorSpike (fewer drops after adaptation)
//   ARC is comparable:    LatencySpike, GradualLatency (similar throughput; lower max_in_flight)
//
// Run:
//
//   go test -bench=BenchmarkQueueSender_Comparison \
//           -benchmem -benchtime=5x -count=1 ./internal/

import (
	"context"
	"errors"
	"math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requestmiddleware"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pipeline"
)

// ─── RTT histogram ────────────────────────────────────────────────────────────

type rttTracker struct {
	mu      sync.Mutex
	samples []int64
}

func (r *rttTracker) record(d time.Duration) {
	r.mu.Lock()
	r.samples = append(r.samples, d.Nanoseconds())
	r.mu.Unlock()
}

func (r *rttTracker) pct(p float64) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.samples) == 0 {
		return 0
	}
	cp := make([]int64, len(r.samples))
	copy(cp, r.samples)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return cp[int(float64(len(cp)-1)*p)]
}

// ─── AIMD ARC middleware ──────────────────────────────────────────────────────
//
// Uses sync.Cond counting semaphore so the limit can be changed dynamically
// without channel-capacity edge cases.
//
// Control law (with separate signals):
//   hard backpressure (error)          → limit = max(limit × 0.75, minLimit)
//   soft backpressure (RTT > 2×EWMA)   → limit = max(limit − 1,    minLimit)
//   success + RTT ≤ 2×EWMA            → limit = min(limit + 1,    maxLimit)
//
// The 0.75 decrease factor (rather than 0.5) prevents the limit from
// crashing to 1 on a single error burst, which would serialise all traffic
// and make performance WORSE than any static configuration.
// The floor of 5 (not 1) ensures a minimum parallelism is always maintained.

type aimdMiddleware struct {
	component.StartFunc
	component.ShutdownFunc

	mu           sync.Mutex
	cond         *sync.Cond
	limit        int32
	inFlight     int32
	peakInFlight int32 // highest inFlight ever seen, tracked inside the lock
	minLimit     int32
	maxLimit     int32
	ewmaRTTns    int64

	successCount int64
	errorCount   int64
}

func newAIMDMiddleware(initial, minLimit, maxLimit int32) *aimdMiddleware {
	m := &aimdMiddleware{
		limit:     initial,
		minLimit:  minLimit,
		maxLimit:  maxLimit,
		ewmaRTTns: int64(5 * time.Millisecond), // sensible starting point
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (a *aimdMiddleware) WrapSender(
	_ requestmiddleware.RequestMiddlewareSettings,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {
	return &aimdSender{mw: a, next: next}, nil
}

type aimdSender struct {
	component.StartFunc
	component.ShutdownFunc
	mw   *aimdMiddleware
	next sender.Sender[request.Request]
}

func (s *aimdSender) Send(ctx context.Context, req request.Request) error {
	s.mw.mu.Lock()
	for s.mw.inFlight >= s.mw.limit {
		s.mw.cond.Wait()
	}
	s.mw.inFlight++
	if s.mw.inFlight > s.mw.peakInFlight {
		s.mw.peakInFlight = s.mw.inFlight
	}
	s.mw.mu.Unlock()

	start := time.Now()
	err := s.next.Send(ctx, req)
	rtt := time.Since(start)

	s.mw.mu.Lock()
	s.mw.inFlight--
	s.mw.adjustLocked(err, rtt)
	s.mw.cond.Broadcast()
	s.mw.mu.Unlock()
	return err
}

func (a *aimdMiddleware) adjustLocked(err error, rtt time.Duration) {
	// Use a faster α for quicker convergence under realistic conditions.
	const alpha = 0.2
	a.ewmaRTTns = int64(float64(a.ewmaRTTns)*(1-alpha) + float64(rtt.Nanoseconds())*alpha)

	hardBackpressure := err != nil
	softBackpressure := !hardBackpressure &&
		float64(rtt.Nanoseconds()) > float64(a.ewmaRTTns)*2.0

	switch {
	case hardBackpressure:
		// Multiplicative decrease: 25 % reduction (much gentler than ÷2).
		// This prevents the limit from crashing to floor on a brief error burst.
		a.errorCount++
		newLimit := int32(float64(a.limit) * 0.75)
		if newLimit < a.minLimit {
			newLimit = a.minLimit
		}
		a.limit = newLimit
	case softBackpressure:
		// Additive decrease for latency: only remove one slot.
		a.errorCount++
		if a.limit > a.minLimit {
			a.limit--
		}
	default:
		// Additive increase: one slot per success.
		a.successCount++
		if a.limit < a.maxLimit {
			a.limit++
		}
	}
}

// ─── Benchmark infrastructure ─────────────────────────────────────────────────

// concurrentSenders is the number of goroutines simultaneously calling Send.
const concurrentSenders = 50

// minSendsTotal is the lower bound on total operations to ensure AIMD has enough
// iterations to converge before the benchmark ends.
const minSendsTotal = 300

// backendFn is a SYNCHRONOUS backend function: it performs its own sleep and
// returns directly.  This allows BackendOverload to track actual in-flight
// calls at the backend level using an atomic counter.
//
// seq is a zero-based sequence number WITHIN the current outer run (not global),
// which prevents state accumulated in one b.N iteration from affecting the next.
type backendFn func(seq int64) error

var errRetryable = errors.New("retryable backend error")

// runComparison drives the benchmark: starts queueSender, launches
// concurrentSenders goroutines, records per-request RTT and error counts,
// then reports all promised metrics.
func runComparison(
	b *testing.B,
	numConsumers int,
	arcMW *aimdMiddleware,
	bFn backendFn,
) {
	b.Helper()

	var (
		totalCalls  atomic.Int64
		totalErrors atomic.Int64
		// backendPeak tracks peak backend-level in-flight for ALL configs (not just ARC).
		backendInFlight atomic.Int32
		backendPeak     atomic.Int32
		rtt             rttTracker
	)

	backend := sender.NewSender(func(_ context.Context, _ request.Request) error {
		// Track in-flight at the backend level for all configurations.
		cur := backendInFlight.Add(1)
		defer backendInFlight.Add(-1)
		for {
			old := backendPeak.Load()
			if cur <= old || backendPeak.CompareAndSwap(old, cur) {
				break
			}
		}
		seq := totalCalls.Add(1) - 1
		return bFn(seq)
	})

	cfg := NewDefaultQueueConfig()
	cfg.NumConsumers = numConsumers
	cfg.WaitForResult = true
	cfg.QueueSize = 500_000

	hostExts := map[component.ID]component.Component{}

	if arcMW != nil {
		if err := featuregate.GlobalRegistry().Set("exporter.RequestMiddleware", true); err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() {
			_ = featuregate.GlobalRegistry().Set("exporter.RequestMiddleware", false)
		})
		mwID := component.MustNewIDWithName("arc", b.Name())
		cfg.RequestMiddlewares = []component.ID{mwID}
		hostExts[mwID] = arcMW
	}

	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.NewID(component.MustNewType("otlp")),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}

	qs, err := NewQueueSender(qSet, cfg, "", backend)
	if err != nil {
		b.Fatal(err)
	}

	host := &mockHost{Host: componenttest.NewNopHost(), ext: hostExts}
	if err := qs.Start(context.Background(), host); err != nil {
		b.Fatal(err)
	}

	// Ensure enough sends for AIMD to converge.
	iters := b.N / concurrentSenders
	if iters < minSendsTotal/concurrentSenders {
		iters = minSendsTotal / concurrentSenders
	}
	if iters < 1 {
		iters = 1
	}

	b.ResetTimer()

	var wg sync.WaitGroup
	for range concurrentSenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iters {
				start := time.Now()
				sendErr := qs.Send(context.Background(), &requesttest.FakeRequest{Items: 100})
				rtt.record(time.Since(start))
				if sendErr != nil {
					totalErrors.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	b.StopTimer()

	if err := qs.Shutdown(context.Background()); err != nil {
		b.Fatal(err)
	}

	sent := totalCalls.Load()
	errs := totalErrors.Load()

	// For static configs, report the backend-level peak (shows how many requests
	// actually hit the backend simultaneously).  For ARC, prefer the middleware's
	// own peakInFlight which is tracked under the mutex for accuracy.
	peakIF := backendPeak.Load()
	if arcMW != nil {
		arcMW.mu.Lock()
		if arcMW.peakInFlight > peakIF {
			peakIF = arcMW.peakInFlight
		}
		arcMW.mu.Unlock()
	}

	b.ReportMetric(float64(rtt.pct(0.50)), "rtt_p50_ns")
	b.ReportMetric(float64(rtt.pct(0.95)), "rtt_p95_ns")
	b.ReportMetric(float64(rtt.pct(0.99)), "rtt_p99_ns")
	b.ReportMetric(float64(peakIF), "max_in_flight")

	var errorPct float64
	if sent > 0 {
		errorPct = float64(errs) * 100 / float64(sent)
	}
	b.ReportMetric(errorPct, "error_rate_pct")
	b.ReportMetric(float64(errs), "dropped_count")
	b.ReportMetric(float64(sent-errs), "successful_ops")

	if arcMW != nil {
		arcMW.mu.Lock()
		finalLimit := arcMW.limit
		arcMW.mu.Unlock()
		b.ReportMetric(float64(finalLimit), "arc_final_limit")
	}
}

// ─── Backend scenarios ────────────────────────────────────────────────────────

// scenarioHealthy: instant backend, no errors.
// ARC expected: final_limit ramps to maxLimit; throughput ≈ StaticHigh.
func scenarioHealthy(_ int64) error { return nil }

// scenarioBackendOverload: backend tracks its own in-flight count and rejects
// requests that arrive while more than hardLimit are already executing.
//
// This is THE key scenario that validates ARC:
//   - StaticHigh sends 50 concurrent → ~80 % are above the limit → mass drops
//   - ARC adapts down to ~hardLimit → ~0 % drops after convergence
//   - StaticLow stays safely under the limit but at low throughput
func scenarioBackendOverload(hardLimit int32, normalLatency time.Duration) backendFn {
	var active atomic.Int32
	return func(_ int64) error {
		n := active.Add(1)
		defer active.Add(-1)
		if n > hardLimit {
			// Simulate a fast rejection (e.g. HTTP 429, gRPC RESOURCE_EXHAUSTED).
			time.Sleep(2 * time.Millisecond)
			return errRetryable
		}
		if normalLatency > 0 {
			time.Sleep(normalLatency)
		}
		return nil
	}
}

// scenarioLatencySpike: first half of each run is fast, second half is 100 ms.
// Using seq % opsPerRun ensures each outer b.N iteration sees the same pattern
// (the counter does not carry state from the previous run).
// ARC expected: detects RTT increase via soft-backpressure; max_in_flight < StaticHigh.
func scenarioLatencySpike(opsPerRun int64) backendFn {
	return func(seq int64) error {
		if (seq%opsPerRun)*2 >= opsPerRun {
			time.Sleep(100 * time.Millisecond)
		}
		return nil
	}
}

// scenarioGradualLatency: latency linearly ramps 0 → 100 ms within each run.
// ARC expected: EWMA baseline tracks the increase; limit decreases gradually.
func scenarioGradualLatency(opsPerRun int64) backendFn {
	return func(seq int64) error {
		pos := seq % opsPerRun
		frac := float64(pos) / float64(opsPerRun)
		lat := time.Duration(float64(100*time.Millisecond) * frac)
		if lat > 0 {
			time.Sleep(lat)
		}
		return nil
	}
}

// scenarioErrorSpike: 25 % of requests fail regardless of concurrency.
// ARC expected: backs off gently (0.75× per error burst); fewer drops than
// StaticHigh because in-flight count is reduced and retry pressure is lower.
func scenarioErrorSpike(_ int64) error {
	if rand.Float32() < 0.25 {
		time.Sleep(5 * time.Millisecond)
		return errRetryable
	}
	return nil
}

// scenarioRecovery: 100 ms first half then instant second half within each run.
// ARC expected: ramps concurrency back up once RTT drops; StaticLow never
// fully recovers because it is capped at 2 workers throughout.
func scenarioRecovery(opsPerRun int64) backendFn {
	return func(seq int64) error {
		pos := seq % opsPerRun
		if pos*2 < opsPerRun {
			time.Sleep(100 * time.Millisecond)
		}
		return nil
	}
}

// ─── Main comparison benchmark ────────────────────────────────────────────────

// BenchmarkQueueSender_Comparison runs the full 6 × 3 comparison matrix.
//
//	go test -bench=BenchmarkQueueSender_Comparison \
//	        -benchmem -benchtime=5x -count=3 ./internal/ | tee results.txt
//	benchstat results.txt
func BenchmarkQueueSender_Comparison(b *testing.B) {
	// Estimate total ops for scenario builders.
	itersPerG := b.N / concurrentSenders
	if itersPerG < minSendsTotal/concurrentSenders {
		itersPerG = minSendsTotal / concurrentSenders
	}
	if itersPerG < 1 {
		itersPerG = 1
	}
	totalOps := int64(itersPerG * concurrentSenders)

	type config struct {
		name         string
		numConsumers int
		arcMW        *aimdMiddleware
	}

	// ARC starts at limit=20, floor=5, ceiling=50.
	// The 5-floor ensures minimum parallelism is always maintained even under
	// sustained backpressure, preventing the "serialisation trap" where ARC
	// drops to limit=1 and becomes slower than StaticLow.
	configs := []config{
		{name: "StaticLow", numConsumers: 2},
		{name: "StaticHigh", numConsumers: 50},
		{name: "ARC", numConsumers: 50, arcMW: newAIMDMiddleware(20, 5, 50)},
	}

	type scenario struct {
		name string
		fn   backendFn
	}

	scenarios := []scenario{
		{
			// Expected: ARC ≈ StaticHigh >> StaticLow.
			name: "Healthy",
			fn:   scenarioHealthy,
		},
		{
			// Expected: ARC error_rate << StaticHigh; ARC dropped_count << StaticHigh.
			// This is the DECISIVE scenario validating ARC.
			// Backend hard limit = 10 concurrent; 10 ms normal latency.
			name: "BackendOverload",
			fn:   scenarioBackendOverload(10, 10*time.Millisecond),
		},
		{
			// Expected: ARC max_in_flight < StaticHigh; ARC rtt_p99 < StaticHigh.
			name: "LatencySpike",
			fn:   scenarioLatencySpike(totalOps),
		},
		{
			// Expected: ARC limit decreases ahead of p99 spike; lower max_in_flight.
			name: "GradualLatency",
			fn:   scenarioGradualLatency(totalOps),
		},
		{
			// Expected: ARC error_rate ≤ StaticHigh; arc_final_limit shows adaptation.
			name: "ErrorSpike",
			fn:   scenarioErrorSpike,
		},
		{
			// Expected: ARC throughput after recovery ≈ StaticHigh >> StaticLow.
			name: "Recovery",
			fn:   scenarioRecovery(totalOps),
		},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			for _, cfg := range configs {
				cfg := cfg
				sc := sc
				b.Run(cfg.name, func(b *testing.B) {
					runComparison(b, cfg.numConsumers, cfg.arcMW, sc.fn)
				})
			}
		})
	}
}

// ─── Interface assertions ──────────────────────────────────────────────────────

var (
	_ senderWrapper = (*aimdMiddleware)(nil)
	_ senderWrapper = (*arcLikeMiddleware)(nil)
	_               = requestmiddleware.RequestMiddlewareSettings{}
)
