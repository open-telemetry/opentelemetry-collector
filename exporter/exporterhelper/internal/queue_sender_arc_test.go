// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// arcLikeMiddleware is a minimal ARC-compatible middleware that demonstrates
// how an adaptive concurrency controller would implement the RequestMiddleware interface.
// This is a test stub, not a full ARC implementation.
type arcLikeMiddleware struct {
	component.StartFunc
	component.ShutdownFunc

	// Concurrency control state
	maxConcurrency int32
	inFlight       atomic.Int32
	permits        chan struct{}

	// Observability state (for testing)
	successCount atomic.Int64
	errorCount   atomic.Int64
}

// newARCLikeMiddleware creates a middleware with the given max concurrency limit.
func newARCLikeMiddleware(maxConcurrency int32) *arcLikeMiddleware {
	return &arcLikeMiddleware{
		maxConcurrency: maxConcurrency,
		permits:        make(chan struct{}, maxConcurrency),
		StartFunc:      component.StartFunc(func(context.Context, component.Host) error { return nil }),
		ShutdownFunc:   component.ShutdownFunc(func(context.Context) error { return nil }),
	}
}

// WrapSender implements the RequestMiddleware interface.
// It wraps the next sender with concurrency control logic.
func (a *arcLikeMiddleware) WrapSender(_ requestmiddleware.RequestMiddlewareSettings, next sender.Sender[request.Request]) (sender.Sender[request.Request], error) {
	return &arcWrapper{
		middleware: a,
		next:       next,
	}, nil
}

// arcWrapper wraps the sender with ARC-like concurrency control.
type arcWrapper struct {
	component.StartFunc
	component.ShutdownFunc
	middleware *arcLikeMiddleware
	next       sender.Sender[request.Request]
}

// Send acquires a permit before sending and releases it after.
func (w *arcWrapper) Send(ctx context.Context, req request.Request) error {
	// Acquire permit (blocks if at max concurrency)
	select {
	case w.middleware.permits <- struct{}{}:
		defer func() { <-w.middleware.permits }()
		w.middleware.inFlight.Add(1)
		defer w.middleware.inFlight.Add(-1)
	case <-ctx.Done():
		return ctx.Err()
	}

	// Send the request
	err := w.next.Send(ctx, req)

	// Record outcome for adaptive decisions (in a real ARC, this would adjust concurrency)
	if err != nil {
		w.middleware.errorCount.Add(1)
	} else {
		w.middleware.successCount.Add(1)
	}

	return err
}

func TestQueueSender_ARCLikeMiddleware(t *testing.T) {
	// Enable the feature gate for this test
	require.NoError(t, featuregate.GlobalRegistry().Set("pkg.exporterhelper.RequestMiddleware", true))
	defer func() {
		_ = featuregate.GlobalRegistry().Set("pkg.exporterhelper.RequestMiddleware", false)
	}()

	mwID := component.MustNewIDWithName("arc", "test")
	arcMW := newARCLikeMiddleware(2) // Max concurrency of 2

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mwID: arcMW,
		},
	}

	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.RequestMiddlewares = []component.ID{mwID}
	qCfg.WaitForResult = true // Make test deterministic

	// Track send calls
	var sendCalls atomic.Int64
	nextSender := sender.NewSender(func(_ context.Context, _ request.Request) error {
		sendCalls.Add(1)
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))

	// Send multiple requests concurrently
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = qs.Send(ctx, &requesttest.FakeRequest{Items: 10})
		}()
	}
	wg.Wait()

	require.NoError(t, qs.Shutdown(context.Background()))

	// Verify that all requests were sent
	assert.Equal(t, int64(5), sendCalls.Load())
	// Verify that ARC observed the requests
	assert.Equal(t, int64(5), arcMW.successCount.Load())
	assert.Equal(t, int64(0), arcMW.errorCount.Load())
}

func TestQueueSender_ARCLikeMiddleware_ErrorPropagation(t *testing.T) {
	require.NoError(t, featuregate.GlobalRegistry().Set("pkg.exporterhelper.RequestMiddleware", true))
	defer func() {
		_ = featuregate.GlobalRegistry().Set("pkg.exporterhelper.RequestMiddleware", false)
	}()

	mwID := component.MustNewIDWithName("arc", "test")
	arcMW := newARCLikeMiddleware(2)

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mwID: arcMW,
		},
	}

	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.RequestMiddlewares = []component.ID{mwID}
	qCfg.WaitForResult = true

	expectedErr := errors.New("backend error")
	nextSender := sender.NewSender(func(_ context.Context, _ request.Request) error {
		return expectedErr
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))

	err = qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10})
	require.Error(t, err)
	assert.Same(t, expectedErr, err)

	require.NoError(t, qs.Shutdown(context.Background()))

	// Verify that ARC observed the error
	assert.Equal(t, int64(1), arcMW.errorCount.Load())
}

func TestQueueSender_ARCLikeMiddleware_ConcurrencyLimit(t *testing.T) {
	require.NoError(t, featuregate.GlobalRegistry().Set("pkg.exporterhelper.RequestMiddleware", true))
	defer func() {
		_ = featuregate.GlobalRegistry().Set("pkg.exporterhelper.RequestMiddleware", false)
	}()

	mwID := component.MustNewIDWithName("arc", "test")
	arcMW := newARCLikeMiddleware(2) // Max concurrency of 2

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mwID: arcMW,
		},
	}

	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.RequestMiddlewares = []component.ID{mwID}
	qCfg.WaitForResult = true

	// Slow sender to verify concurrency limiting.
	// We track inFlight directly from the middleware's own counter; the arcWrapper already
	// increments it before calling Send, so we just snapshot it here.
	var maxInFlight atomic.Int32
	nextSender := sender.NewSender(func(_ context.Context, _ request.Request) error {
		// arcWrapper has already incremented arcMW.inFlight before calling us.
		current := arcMW.inFlight.Load()
		for {
			old := maxInFlight.Load()
			if current <= old || maxInFlight.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))

	// Send more requests than the concurrency limit
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = qs.Send(ctx, &requesttest.FakeRequest{Items: 10})
		}()
	}
	wg.Wait()

	require.NoError(t, qs.Shutdown(context.Background()))

	// Verify that concurrency was limited
	assert.Equal(t, int32(2), maxInFlight.Load(), "max in-flight should not exceed concurrency limit")
}
