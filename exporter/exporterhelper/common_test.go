// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

var (
	defaultType     = component.MustNewType("test")
	defaultID       = component.NewID(defaultType)
	defaultSettings = func() exporter.CreateSettings {
		set := exportertest.NewNopCreateSettings()
		set.ID = defaultID
		return set
	}()
)

func newNoopObsrepSender(*ObsReport) requestSender {
	return &baseRequestSender{}
}

func TestBaseExporter(t *testing.T) {
	be, err := newBaseExporter(defaultSettings, defaultType, newNoopObsrepSender)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestBaseExporterWithOptions(t *testing.T) {
	want := errors.New("my error")
	be, err := newBaseExporter(
		defaultSettings, defaultType, newNoopObsrepSender,
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithTimeout(NewDefaultTimeoutSettings()),
	)
	require.NoError(t, err)
	require.Equal(t, want, be.Start(context.Background(), componenttest.NewNopHost()))
	require.Equal(t, want, be.Shutdown(context.Background()))
}

func checkStatus(t *testing.T, sd sdktrace.ReadOnlySpan, err error) {
	if err != nil {
		require.Equal(t, codes.Error, sd.Status().Code, "SpanData %v", sd)
		require.Equal(t, err.Error(), sd.Status().Description, "SpanData %v", sd)
	} else {
		require.Equal(t, codes.Unset, sd.Status().Code, "SpanData %v", sd)
	}
}

func TestQueueOptionsWithRequestExporter(t *testing.T) {
	bs, err := newBaseExporter(exportertest.NewNopCreateSettings(), defaultType, newNoopObsrepSender,
		WithRetry(configretry.NewDefaultBackOffConfig()))
	require.Nil(t, err)
	require.Nil(t, bs.marshaler)
	require.Nil(t, bs.unmarshaler)
	_, err = newBaseExporter(exportertest.NewNopCreateSettings(), defaultType, newNoopObsrepSender,
		WithRetry(configretry.NewDefaultBackOffConfig()), WithQueue(NewDefaultQueueSettings()))
	require.Error(t, err)

	_, err = newBaseExporter(exportertest.NewNopCreateSettings(), defaultType, newNoopObsrepSender,
		withMarshaler(mockRequestMarshaler), withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(configretry.NewDefaultBackOffConfig()),
		WithRequestQueue(exporterqueue.NewDefaultConfig(), exporterqueue.NewMemoryQueueFactory[Request]()))
	require.Error(t, err)
}

func TestBaseExporterLogging(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = false
	bs, err := newBaseExporter(set, defaultType, newNoopObsrepSender, WithRetry(rCfg))
	require.Nil(t, err)
	sendErr := bs.send(context.Background(), newErrorRequest())
	require.Error(t, sendErr)

	require.Len(t, observed.FilterLevelExact(zap.ErrorLevel).All(), 1)
}
