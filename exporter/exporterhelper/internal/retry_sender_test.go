// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

type fakeRefCounter struct {
	mu         sync.Mutex
	unrefCount int
	unreffed   []request.Request
}

func (f *fakeRefCounter) Ref(request.Request) {}

func (f *fakeRefCounter) Unref(req request.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unrefCount++
	f.unreffed = append(f.unreffed, req)
}

func TestRetrySenderDropOnPermanentError(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	sink := requesttest.NewSink()
	expErr := consumererror.NewPermanent(errors.New("bad data"))
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export), nil)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetExportErr(expErr)
	require.ErrorIs(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 2}), expErr)
	sink.SetExportErr(expErr)
	require.ErrorIs(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 3}), expErr)
	assert.Equal(t, 0, sink.ItemsCount())
	assert.Equal(t, 0, sink.RequestsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderSimpleRetry(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	sink := requesttest.NewSink()
	expErr := errors.New("transient error")
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export), nil)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetExportErr(expErr)
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 2}))
	assert.Equal(t, 2, sink.ItemsCount())
	assert.Equal(t, 1, sink.RequestsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderRetryPartial(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	sink := requesttest.NewSink()
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export), nil)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 5, Partial: 3}))
	assert.Equal(t, 5, sink.ItemsCount())
	assert.Equal(t, 2, sink.RequestsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderMaxElapsedTime(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 100 * time.Millisecond
	expErr := errors.New("transient error")
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(func(context.Context, request.Request) error { return expErr }), nil)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.ErrorIs(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 2}), expErr)
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderThrottleError(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 10 * time.Millisecond
	sink := requesttest.NewSink()
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export), nil)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	retry := fmt.Errorf("wrappe error: %w", NewThrottleRetry(errors.New("throttle error"), 100*time.Millisecond))
	start := time.Now()
	sink.SetExportErr(retry)
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 5}))
	// The initial backoff is 10ms, but because of the throttle this should wait at least 100ms.
	assert.Less(t, 100*time.Millisecond, time.Since(start))
	assert.Equal(t, 5, sink.ItemsCount())
	assert.Equal(t, 1, sink.RequestsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderWithContextTimeout(t *testing.T) {
	const testTimeout = 10 * time.Second
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = true
	// First attempt after 100ms is attempted
	rCfg.InitialInterval = 100 * time.Millisecond
	rCfg.RandomizationFactor = 0
	// Second attempt is at twice the testTimeout
	rCfg.Multiplier = float64(2 * testTimeout / rCfg.InitialInterval)
	set := exportertest.NewNopSettings(exportertest.NopType)
	logger, observed := observer.New(zap.InfoLevel)
	set.Logger = zap.New(logger)
	rs := newRetrySender(rCfg, set, sender.NewSender(func(context.Context, request.Request) error { return errors.New("transient error") }), nil)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	require.ErrorContains(t,
		rs.Send(ctx, &requesttest.FakeRequest{Items: 2}),
		"request will be cancelled before next retry: transient error")
	assert.Len(t, observed.All(), 1)
	assert.Equal(t, "Exporting failed. Will retry the request after interval.", observed.All()[0].Message)
	require.Less(t, time.Since(start), testTimeout/2)
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderWithCancelledContext(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = true
	// First attempt after 1s is attempted
	rCfg.InitialInterval = 1 * time.Second
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(func(context.Context, request.Request) error { return errors.New("transient error") }), nil)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	start := time.Now()
	ctx, cancel := context.WithCancelCause(context.Background())
	go func() {
		<-time.After(100 * time.Millisecond)
		cancel(errors.New("my reason"))
	}()
	require.ErrorContains(t,
		rs.Send(ctx, &requesttest.FakeRequest{Items: 2}),
		"request is cancelled or timed out: transient error")
	require.Less(t, time.Since(start), 1*time.Second)
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderRetryPartialUnref(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	sink := requesttest.NewSink()
	rc := &fakeRefCounter{}
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export), rc)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 5, Partial: 3}))
	assert.Equal(t, 5, sink.ItemsCount())
	assert.Equal(t, 2, sink.RequestsCount())
	// The replacement request created by OnError should be Unref'd exactly once.
	assert.Equal(t, 1, rc.unrefCount)
	assert.Equal(t, 3, rc.unreffed[0].ItemsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderRetryMultiplePartialUnref(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	rc := &fakeRefCounter{}
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(func(_ context.Context, req request.Request) error {
		if req.ItemsCount() > 1 {
			return requesttest.NewPartialError(&requesttest.FakeRequest{Items: req.ItemsCount() - 1})
		}
		return nil
	}), rc)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))
	// Items=3 → partial(2) → partial(1) → success. Two replacements.
	// req_B(Items=2) is Unref'd in the loop, req_C(Items=1) is Unref'd in defer.
	assert.Equal(t, 2, rc.unrefCount)
	assert.Equal(t, 2, rc.unreffed[0].ItemsCount())
	assert.Equal(t, 1, rc.unreffed[1].ItemsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderSimpleRetryNoUnref(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	sink := requesttest.NewSink()
	rc := &fakeRefCounter{}
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export), rc)
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 2}))
	assert.Equal(t, 2, sink.ItemsCount())
	// No replacement happened, so Unref should not be called.
	assert.Equal(t, 0, rc.unrefCount)
	require.NoError(t, rs.Shutdown(context.Background()))
}
