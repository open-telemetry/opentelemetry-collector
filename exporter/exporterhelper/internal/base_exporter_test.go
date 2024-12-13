// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	defaultType     = component.MustNewType("test")
	defaultSignal   = pipeline.SignalMetrics
	defaultID       = component.NewID(defaultType)
	defaultSettings = func() exporter.Settings {
		set := exportertest.NewNopSettings()
		set.ID = defaultID
		return set
	}()
)

func newNoopObsrepSender(*ObsReport) RequestSender {
	return &BaseRequestSender{}
}

func TestBaseExporter(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, be.Shutdown(context.Background()))
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestBaseExporterWithOptions(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			want := errors.New("my error")
			be, err := NewBaseExporter(
				defaultSettings, defaultSignal, newNoopObsrepSender,
				WithStart(func(context.Context, component.Host) error { return want }),
				WithShutdown(func(context.Context) error { return want }),
				WithTimeout(NewDefaultTimeoutConfig()),
			)
			require.NoError(t, err)
			require.Equal(t, want, be.Start(context.Background(), componenttest.NewNopHost()))
			require.Equal(t, want, be.Shutdown(context.Background()))
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestQueueOptionsWithRequestExporter(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			bs, err := NewBaseExporter(exportertest.NewNopSettings(), defaultSignal, newNoopObsrepSender,
				WithRetry(configretry.NewDefaultBackOffConfig()))
			require.NoError(t, err)
			require.Nil(t, bs.Marshaler)
			require.Nil(t, bs.Unmarshaler)
			_, err = NewBaseExporter(exportertest.NewNopSettings(), defaultSignal, newNoopObsrepSender,
				WithRetry(configretry.NewDefaultBackOffConfig()), WithQueue(NewDefaultQueueConfig()))
			require.Error(t, err)

			_, err = NewBaseExporter(exportertest.NewNopSettings(), defaultSignal, newNoopObsrepSender,
				WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
				WithRetry(configretry.NewDefaultBackOffConfig()),
				WithRequestQueue(exporterqueue.NewDefaultConfig(), exporterqueue.NewMemoryQueueFactory[internal.Request]()))
			require.Error(t, err)
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestBaseExporterLogging(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			set := exportertest.NewNopSettings()
			logger, observed := observer.New(zap.DebugLevel)
			set.Logger = zap.New(logger)
			rCfg := configretry.NewDefaultBackOffConfig()
			rCfg.Enabled = false
			bs, err := NewBaseExporter(set, defaultSignal, newNoopObsrepSender, WithRetry(rCfg))
			require.NoError(t, err)
			sendErr := bs.Send(context.Background(), newErrorRequest())
			require.Error(t, sendErr)

			require.Len(t, observed.FilterLevelExact(zap.ErrorLevel).All(), 1)
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}
