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
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestBaseExporter(t *testing.T) {
	be, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestBaseExporterWithOptions(t *testing.T) {
	want := errors.New("my error")
	be, err := NewBaseExporter(
		exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithTimeout(NewDefaultTimeoutConfig()),
	)
	require.NoError(t, err)
	require.Equal(t, want, be.Start(context.Background(), componenttest.NewNopHost()))
	require.Equal(t, want, be.Shutdown(context.Background()))
}

func TestQueueOptionsWithRequestExporter(t *testing.T) {
	bs, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithRetry(configretry.NewDefaultBackOffConfig()))
	require.NoError(t, err)
	require.Nil(t, bs.queueBatchSettings.Encoding)
	_, err = NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithRetry(configretry.NewDefaultBackOffConfig()), WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.Error(t, err)

	qCfg := NewDefaultQueueConfig()
	storageID := component.NewID(component.MustNewType("test"))
	qCfg.StorageID = &storageID
	_, err = NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithRetry(configretry.NewDefaultBackOffConfig()),
		WithQueueBatch(configoptional.Some(qCfg), queuebatch.Settings[request.Request]{}))
	require.Error(t, err)
}

func TestBaseExporterLogging(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = false
	qCfg := NewDefaultQueueConfig()
	qCfg.WaitForResult = true
	bs, err := NewBaseExporter(set, pipeline.SignalMetrics, errExport,
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithQueue(configoptional.Some(qCfg)),
		WithRetry(rCfg))
	require.NoError(t, err)
	require.NoError(t, bs.Start(context.Background(), componenttest.NewNopHost()))
	sendErr := bs.Send(context.Background(), &requesttest.FakeRequest{Items: 2})
	require.Error(t, sendErr)

	errorLogs := observed.FilterLevelExact(zap.ErrorLevel).All()
	require.Len(t, errorLogs, 2)
	assert.Contains(t, errorLogs[0].Message, "Exporting failed. Dropping data.")
	assert.Equal(t, "my error", errorLogs[0].ContextMap()["error"])
	assert.Contains(t, errorLogs[1].Message, "Exporting failed. Rejecting data.")
	assert.Equal(t, "my error", errorLogs[1].ContextMap()["error"])
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
				WithQueueBatchSettings(newFakeQueueBatch()),
				func() Option {
					return WithQueue(configoptional.None[queuebatch.Config]())
				}(),
			},
		},
		{
			name: "WithRequestQueue",
			queueOptions: []Option{
				func() Option {
					return WithQueueBatch(configoptional.None[queuebatch.Config](), newFakeQueueBatch())
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := exportertest.NewNopSettings(exportertest.NopType)
			logger, observed := observer.New(zap.ErrorLevel)
			set.Logger = zap.New(logger)
			be, err := NewBaseExporter(set, pipeline.SignalLogs, errExport, tt.queueOptions...)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			mockR := &requesttest.FakeRequest{Items: 2}
			require.Error(t, be.Send(context.Background(), mockR))
			assert.Len(t, observed.All(), 1)
			assert.Equal(t, "Exporting failed. Rejecting data. Try enabling sending_queue to survive temporary failures.", observed.All()[0].Message)
			require.NoError(t, be.Shutdown(context.Background()))
		})
	}
}

func errExport(context.Context, request.Request) error {
	return errors.New("my error")
}

func noopExport(context.Context, request.Request) error {
	return nil
}

func newFakeQueueBatch() queuebatch.Settings[request.Request] {
	return queuebatch.Settings[request.Request]{
		Encoding: fakeEncoding{},
	}
}

type fakeEncoding struct{}

func (f fakeEncoding) Marshal(context.Context, request.Request) ([]byte, error) {
	return []byte("mockRequest"), nil
}

func (f fakeEncoding) Unmarshal([]byte) (context.Context, request.Request, error) {
	return context.Background(), &requesttest.FakeRequest{}, nil
}
