// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"fmt"
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

func TestRetrySenderDropOnPermanentError(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	sink := requesttest.NewSink()
	expErr := consumererror.NewPermanent(errors.New("bad data"))
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export))
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
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export))
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
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export))
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
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(func(context.Context, request.Request) error { return expErr }))
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.ErrorIs(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 2}), expErr)
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderThrottleError(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 10 * time.Millisecond
	sink := requesttest.NewSink()
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(sink.Export))
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
	rs := newRetrySender(rCfg, set, sender.NewSender(func(context.Context, request.Request) error { return errors.New("transient error") }))
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
	rs := newRetrySender(rCfg, exportertest.NewNopSettings(exportertest.NopType), sender.NewSender(func(context.Context, request.Request) error { return errors.New("transient error") }))
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
