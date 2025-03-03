// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	defaultType     = component.MustNewType("test")
	defaultSignal   = pipeline.SignalMetrics
	defaultID       = component.NewID(defaultType)
	defaultSettings = func() exporter.Settings {
		set := exportertest.NewNopSettings(exportertest.NopType)
		set.ID = defaultID
		return set
	}()
)

func newNoopExportSender() Sender[request.Request] {
	return newSender(func(ctx context.Context, req request.Request) error {
		select {
		case <-ctx.Done():
			return ctx.Err() // Returns the cancellation error
		default:
			return req.Export(ctx)
		}
	})
}

func TestBaseExporter(t *testing.T) {
	be, err := NewBaseExporter(defaultSettings, defaultSignal)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestBaseExporterWithOptions(t *testing.T) {
	want := errors.New("my error")
	be, err := NewBaseExporter(
		defaultSettings, defaultSignal,
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithTimeout(NewDefaultTimeoutConfig()),
	)
	require.NoError(t, err)
	require.Equal(t, want, be.Start(context.Background(), componenttest.NewNopHost()))
	require.Equal(t, want, be.Shutdown(context.Background()))
}

func TestQueueOptionsWithRequestExporter(t *testing.T) {
	bs, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), defaultSignal,
		WithRetry(configretry.NewDefaultBackOffConfig()))
	require.NoError(t, err)
	require.Nil(t, bs.encoding)
	_, err = NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), defaultSignal,
		WithRetry(configretry.NewDefaultBackOffConfig()), WithQueue(exporterqueue.NewDefaultConfig()))
	require.Error(t, err)

	qCfg := exporterqueue.NewDefaultConfig()
	storageID := component.NewID(component.MustNewType("test"))
	qCfg.StorageID = &storageID
	_, err = NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), defaultSignal,
		WithEncoding(newFakeEncoding(&requesttest.FakeRequest{Items: 1})),
		WithRetry(configretry.NewDefaultBackOffConfig()),
		WithRequestQueue(qCfg, nil))
	require.Error(t, err)
}

func TestBaseExporterLogging(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = false
	qCfg := exporterqueue.NewDefaultConfig()
	qCfg.Enabled = false
	bs, err := NewBaseExporter(set, defaultSignal,
		WithRequestQueue(qCfg, newFakeEncoding(&requesttest.FakeRequest{})),
		WithBatcher(exporterbatcher.NewDefaultConfig()),
		WithRetry(rCfg))
	require.NoError(t, err)
	require.NoError(t, bs.Start(context.Background(), componenttest.NewNopHost()))
	sink := requesttest.NewSink()
	sendErr := bs.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink, ExportErr: errors.New("my error")})
	require.Error(t, sendErr)

	require.Len(t, observed.FilterLevelExact(zap.ErrorLevel).All(), 2)
	assert.Contains(t, observed.All()[0].Message, "Exporting failed. Dropping data.")
	assert.Equal(t, "my error", observed.All()[0].ContextMap()["error"])
	assert.Contains(t, observed.All()[1].Message, "Exporting failed. Rejecting data.")
	assert.Equal(t, "my error", observed.All()[1].ContextMap()["error"])
	require.NoError(t, bs.Shutdown(context.Background()))
}

func TestQueueRetryWithDisabledQueue(t *testing.T) {
	tests := []struct {
		name         string
		queueOptions []Option
	}{
		{
			name: "WithQueue",
			queueOptions: []Option{
				WithEncoding(newFakeEncoding(&requesttest.FakeRequest{Items: 1})),
				func() Option {
					qs := exporterqueue.NewDefaultConfig()
					qs.Enabled = false
					return WithQueue(qs)
				}(),
				func() Option {
					bs := exporterbatcher.NewDefaultConfig()
					bs.Enabled = false
					return WithBatcher(bs)
				}(),
			},
		},
		{
			name: "WithRequestQueue",
			queueOptions: []Option{
				func() Option {
					qs := exporterqueue.NewDefaultConfig()
					qs.Enabled = false
					return WithRequestQueue(qs, newFakeEncoding(&requesttest.FakeRequest{Items: 1}))
				}(),
				func() Option {
					bs := exporterbatcher.NewDefaultConfig()
					bs.Enabled = false
					return WithBatcher(bs)
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := exportertest.NewNopSettings(exportertest.NopType)
			logger, observed := observer.New(zap.ErrorLevel)
			set.Logger = zap.New(logger)
			be, err := NewBaseExporter(set, pipeline.SignalLogs, tt.queueOptions...)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			sink := requesttest.NewSink()
			mockR := &requesttest.FakeRequest{Items: 2, Sink: sink, ExportErr: errors.New("some error")}
			require.Error(t, be.Send(context.Background(), mockR))
			assert.Len(t, observed.All(), 1)
			assert.Equal(t, "Exporting failed. Rejecting data. Try enabling sending_queue to survive temporary failures.", observed.All()[0].Message)
			require.NoError(t, be.Shutdown(context.Background()))
			assert.Empty(t, 0, sink.RequestsCount())
		})
	}
}
