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
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func mockRequestUnmarshaler(mr request.Request) exporterqueue.Unmarshaler[request.Request] {
	return func([]byte) (request.Request, error) {
		return mr, nil
	}
}

func mockRequestMarshaler(request.Request) ([]byte, error) {
	return []byte("mockRequest"), nil
}

func TestRetrySenderDropOnPermanentError(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	sink := requesttest.NewSink()
	expErr := consumererror.NewPermanent(errors.New("bad data"))
	rs := newRetrySender(rCfg, exportertest.NewNopSettingsWithType(exportertest.NopType), newNoopExportSender())
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.ErrorIs(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink, ExportErr: expErr}), expErr)
	require.ErrorIs(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 3, Sink: sink, ExportErr: expErr}), expErr)
	assert.Equal(t, int64(0), sink.ItemsCount())
	assert.Equal(t, int64(0), sink.RequestsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderSimpleRetry(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	sink := requesttest.NewSink()
	expErr := errors.New("transient error")
	rs := newRetrySender(rCfg, exportertest.NewNopSettingsWithType(exportertest.NopType), newNoopExportSender())
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink, ExportErr: expErr}))
	assert.Equal(t, int64(2), sink.ItemsCount())
	assert.Equal(t, int64(1), sink.RequestsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderRetryPartial(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	sink := requesttest.NewSink()
	rs := newRetrySender(rCfg, exportertest.NewNopSettingsWithType(exportertest.NopType), newNoopExportSender())
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 5, Sink: sink, Partial: 3}))
	assert.Equal(t, int64(5), sink.ItemsCount())
	assert.Equal(t, int64(2), sink.RequestsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderMaxElapsedTime(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 100 * time.Millisecond
	sink := requesttest.NewSink()
	rs := newRetrySender(rCfg, exportertest.NewNopSettingsWithType(exportertest.NopType), newNoopExportSender())
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	expErr := errors.New("transient error")
	require.ErrorIs(t, rs.Send(context.Background(), newErrorRequest(expErr)), expErr)
	assert.Equal(t, int64(0), sink.ItemsCount())
	assert.Equal(t, int64(0), sink.RequestsCount())
	require.NoError(t, rs.Shutdown(context.Background()))
}

func TestRetrySenderThrottleError(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 10 * time.Millisecond
	sink := requesttest.NewSink()
	rs := newRetrySender(rCfg, exportertest.NewNopSettingsWithType(exportertest.NopType), newNoopExportSender())
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	retry := fmt.Errorf("wrappe error: %w", NewThrottleRetry(errors.New("throttle error"), 100*time.Millisecond))
	start := time.Now()
	require.NoError(t, rs.Send(context.Background(), &requesttest.FakeRequest{Items: 5, Sink: sink, ExportErr: retry}))
	// The initial backoff is 10ms, but because of the throttle this should wait at least 100ms.
	assert.Less(t, 100*time.Millisecond, time.Since(start))
	assert.Equal(t, int64(5), sink.ItemsCount())
	assert.Equal(t, int64(1), sink.RequestsCount())
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
	set := exportertest.NewNopSettingsWithType(exportertest.NopType)
	logger, observed := observer.New(zap.InfoLevel)
	set.Logger = zap.New(logger)
	rs := newRetrySender(rCfg, set, newNoopExportSender())
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	require.ErrorContains(t,
		rs.Send(ctx, newErrorRequest(errors.New("transient error"))),
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
	rs := newRetrySender(rCfg, exportertest.NewNopSettingsWithType(exportertest.NopType), newNoopExportSender())
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))
	start := time.Now()
	ctx, cancel := context.WithCancelCause(context.Background())
	go func() {
		<-time.After(100 * time.Millisecond)
		cancel(errors.New("my reason"))
	}()
	require.ErrorContains(t,
		rs.Send(ctx, newErrorRequest(errors.New("transient error"))),
		"request is cancelled or timed out: transient error")
	require.Less(t, time.Since(start), 1*time.Second)
	require.NoError(t, rs.Shutdown(context.Background()))
}

type mockErrorRequest struct {
	err error
}

func (mer *mockErrorRequest) Export(context.Context) error {
	return mer.err
}

func (mer *mockErrorRequest) OnError(error) request.Request {
	return mer
}

func (mer *mockErrorRequest) ItemsCount() int {
	return 7
}

func (mer *mockErrorRequest) MergeSplit(context.Context, exporterbatcher.MaxSizeConfig, request.Request) ([]request.Request, error) {
	return nil, nil
}

func newErrorRequest(err error) request.Request {
	return &mockErrorRequest{err: err}
}
