// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/arc"

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/pipeline"
)

func TestNewControllerDefaults(t *testing.T) {
	cfg := Config{
		InitialLimit:   0,
		MaxConcurrency: 0,
		DecreaseRatio:  0,
		EwmaAlpha:      0,
		DeviationScale: -1,
	}
	ctrl := NewController(cfg, nil, component.ID{}, pipeline.SignalTraces)

	if got, want := ctrl.CurrentLimit(), DefaultConfig().InitialLimit; got != want {
		t.Fatalf("unexpected InitialLimit: got %d want %d", got, want)
	}
	if ctrl.cfg.MaxConcurrency != DefaultConfig().MaxConcurrency {
		t.Fatalf("MaxConcurrency default not applied")
	}
	if ctrl.cfg.DecreaseRatio != DefaultConfig().DecreaseRatio {
		t.Fatalf("DecreaseRatio default not applied")
	}
	if ctrl.cfg.EwmaAlpha != DefaultConfig().EwmaAlpha {
		t.Fatalf("EwmaAlpha default not applied")
	}
	if ctrl.cfg.DeviationScale != DefaultConfig().DeviationScale {
		t.Fatalf("DeviationScale default not applied")
	}
}

func TestAcquireReleaseSemaphore(t *testing.T) {
	cfg := Config{InitialLimit: 2}
	ctrl := NewController(cfg, nil, component.ID{}, pipeline.SignalTraces)

	ctx := context.Background()
	if !ctrl.Acquire(ctx) {
		t.Fatal("expected first Acquire to succeed")
	}
	if !ctrl.Acquire(ctx) {
		t.Fatal("expected second Acquire to succeed")
	}

	// third attempt should block and then fail with context timeout
	toCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if ctrl.Acquire(toCtx) {
		t.Fatal("expected Acquire with short timeout to fail")
	}

	// release one and acquire should succeed
	ctrl.Release()
	if !ctrl.Acquire(ctx) {
		t.Fatal("expected Acquire to succeed after release")
	}
	// cleanup releases
	ctrl.Release()
	ctrl.Release()
}

func TestStartRequestAndPermits(t *testing.T) {
	cfg := Config{InitialLimit: 3}
	ctrl := NewController(cfg, nil, component.ID{}, pipeline.SignalTraces)

	ctrl.StartRequest()
	ctrl.StartRequest()

	if got := ctrl.PermitsInUse(); got != 2 {
		t.Fatalf("unexpected in-flight: got %d want %d", got, 2)
	}

	// Simulate hitting ceiling
	ctrl.mu.Lock()
	ctrl.st.inFlight = ctrl.st.limit
	ctrl.st.hitCeiling = false
	ctrl.mu.Unlock()

	ctrl.StartRequest()
	ctrl.mu.Lock()
	hit := ctrl.st.hitCeiling
	ctrl.mu.Unlock()
	if !hit {
		t.Fatalf("expected hitCeiling to be true when inFlight >= limit")
	}
}

func TestEarlyBackoffOnColdStart(t *testing.T) {
	cfg := Config{InitialLimit: 10, DecreaseRatio: 0.5}
	ctrl := NewController(cfg, nil, component.ID{}, pipeline.SignalTraces)

	if got := ctrl.CurrentLimit(); got != 10 {
		t.Fatalf("expected initial limit 10, got %d", got)
	}

	// Provide an explicit backpressure while EWMA not initialized.
	ctrl.Feedback(context.Background(), 0, true, true)

	if got := ctrl.CurrentLimit(); got != 5 {
		t.Fatalf("expected early backoff to reduce limit to 5, got %d", got)
	}
}

func TestAdjustIncreaseAndDecrease(t *testing.T) {
	// Additive increase case
	cfg := DefaultConfig()
	cfg.InitialLimit = 1
	ctrl := NewController(cfg, nil, component.ID{}, pipeline.SignalTraces)

	ctrl.mu.Lock()
	// initialize past EWMA so past.initialized() is true
	ctrl.st.past.update(1.0)
	ctrl.st.hitCeiling = true
	ctrl.st.hadPressure = false
	ctrl.st.limit = 1
	ctrl.mu.Unlock()

	ctrl.mu.Lock()
	ctrl.adjust(context.Background(), 1.0)
	ctrl.mu.Unlock()

	if got := ctrl.CurrentLimit(); got != 2 {
		t.Fatalf("expected additive increase to raise limit to 2, got %d", got)
	}

	// Multiplicative decrease case
	cfg2 := DefaultConfig()
	cfg2.InitialLimit = 10
	cfg2.DecreaseRatio = 0.5
	ctrl2 := NewController(cfg2, nil, component.ID{}, pipeline.SignalTraces)

	ctrl2.mu.Lock()
	ctrl2.st.past.update(1.0)
	ctrl2.st.hadPressure = true
	ctrl2.st.limit = 10
	ctrl2.mu.Unlock()

	ctrl2.mu.Lock()
	ctrl2.adjust(context.Background(), 0.0)
	ctrl2.mu.Unlock()

	if got := ctrl2.CurrentLimit(); got != 5 {
		t.Fatalf("expected multiplicative decrease to reduce limit to 5, got %d", got)
	}
}

func TestShrinkSem_ForgetAndRelease(t *testing.T) {
	s := newShrinkSem(2)

	// Use capacity
	require.NoError(t, s.acquire(context.Background()))
	require.NoError(t, s.acquire(context.Background()))
	require.Equal(t, 0, s.avail)

	// Forget 1, adds pending Forget
	s.forget(1)
	require.Equal(t, 0, s.avail)
	require.Equal(t, 1, s.pendingF)

	// Release 1, pays pending Forget
	s.release()
	require.Equal(t, 0, s.avail)
	require.Equal(t, 0, s.pendingF)

	// Next acquire should block
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	require.Error(t, s.acquire(ctx)) // Fails on timeout
	cancel()

	// Release second acquire
	s.release()
	require.Equal(t, 1, s.avail)
	require.Equal(t, 0, s.pendingF)

	// Acquire should now succeed
	require.NoError(t, s.acquire(context.Background()))
	require.Equal(t, 0, s.avail)

	// Forget 2. No avail, one permit outstanding.
	// Both forgets should go to pendingF.
	s.forget(2)
	require.Equal(t, 0, s.avail)
	require.Equal(t, 2, s.pendingF)

	// Release (pays one pending forget)
	s.release()
	require.Equal(t, 0, s.avail)
	require.Equal(t, 1, s.pendingF)
}

func TestShrinkSem_AddAndAcquire(t *testing.T) {
	s := newShrinkSem(1)
	require.NoError(t, s.acquire(context.Background()))
	require.Equal(t, 0, s.avail)

	// Start a waiting acquire in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Must use assert in goroutines
		assert.NoError(t, s.acquire(context.Background()))
		s.release() // release it right away
	}()

	// Allow goroutine to block on acquire
	time.Sleep(10 * time.Millisecond)

	// Add one permit; should wake the waiter
	s.addOne()
	wg.Wait() // ensure goroutine finished

	s.mu.Lock()
	require.Equal(t, 1, s.avail) // the goroutine acquired then released once
	require.Empty(t, s.waiting)
	s.mu.Unlock()

	// Test Add paying pending Forgets
	s.forget(2) // avail=1 -> 0, pendingF=1
	require.Equal(t, 0, s.avail)
	require.Equal(t, 1, s.pendingF)

	s.addOne() // pays pendingF
	require.Equal(t, 0, s.avail)
	require.Equal(t, 0, s.pendingF)

	s.addOne() // increases avail
	require.Equal(t, 1, s.avail)
	require.Equal(t, 0, s.pendingF)
}

func TestController_Shutdown(t *testing.T) {
	// Test that shutdown doesn't panic
	ctrl := NewController(DefaultConfig(), nil, component.ID{}, pipeline.SignalTraces)
	ctrl.Shutdown()

	// Test that shutdown with telemetry doesn't panic
	tel, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	ctrlWithTel := NewController(DefaultConfig(), tel, component.ID{}, pipeline.SignalTraces)
	ctrlWithTel.Shutdown()
}

func TestController_Shutdown_UnblocksWaiters(t *testing.T) {
	ctrl := NewController(Config{InitialLimit: 1}, nil, component.ID{}, pipeline.SignalTraces)

	// Acquire the only permit
	require.True(t, ctrl.Acquire(context.Background()))

	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// This should block
		errCh <- ctrl.sem.acquire(context.Background())
	}()

	// Give the goroutine time to block
	time.Sleep(50 * time.Millisecond)

	// Shutdown the controller, which closes the semaphore
	ctrl.Shutdown()

	// Wait for the goroutine to finish
	wg.Wait()

	// Check that the blocked goroutine received a shutdown error
	err := <-errCh
	require.Error(t, err)
	assert.True(t, experr.IsShutdownErr(err), "error should be a shutdown error")

	// Test that new acquires also fail
	err = ctrl.sem.acquire(context.Background())
	require.Error(t, err)
	assert.True(t, experr.IsShutdownErr(err), "error should be a shutdown error")
}

func TestController_ReleaseWithSample(t *testing.T) {
	cfg := Config{InitialLimit: 1}
	ctrl := NewController(cfg, nil, component.ID{}, pipeline.SignalTraces)

	ctx := context.Background()
	require.True(t, ctrl.Acquire(ctx))
	ctrl.StartRequest()

	// Initialize EWMA
	ctrl.ReleaseWithSample(ctx, 10*time.Millisecond, true, false)
	ctrl.mu.Lock()
	require.True(t, ctrl.st.past.initialized())
	require.Equal(t, 0, ctrl.st.inFlight)
	ctrl.mu.Unlock()
}

func TestController_FeedbackWindowTick(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InitialLimit = 1
	ctrl := NewController(cfg, nil, component.ID{}, pipeline.SignalTraces)

	// Initialize past EWMA
	ctrl.mu.Lock()
	ctrl.st.past.update(1.0)
	ctrl.st.limit = 1
	ctrl.st.nextTick = time.Now() // Force tick on next feedback
	ctrl.st.hitCeiling = true     // Enable additive increase
	ctrl.mu.Unlock()

	// This feedback should trigger a window tick
	ctrl.Feedback(context.Background(), 10*time.Millisecond, true, false)

	ctrl.mu.Lock()
	// Check if adjust ran (limit increased)
	require.Equal(t, 2, ctrl.st.limit)
	// Check if state was reset
	require.Equal(t, 0, ctrl.st.curr.n)
	require.False(t, ctrl.st.hadPressure)
	require.False(t, ctrl.st.hitCeiling)
	require.True(t, ctrl.st.nextTick.After(time.Now())) // Next tick is in the future
	ctrl.mu.Unlock()
}
