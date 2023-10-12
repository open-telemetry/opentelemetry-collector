// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

var (
	defaultID       = component.NewID("test")
	defaultSettings = func() exporter.CreateSettings {
		set := exportertest.NewNopCreateSettings()
		set.ID = defaultID
		return set
	}()
)

func newNoopObsrepSender(_ *ObsReport) requestSender {
	return &baseRequestSender{}
}

func TestBaseExporter(t *testing.T) {
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newNoopObsrepSender)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Shutdown(context.Background()))
	be, err = newBaseExporter(defaultSettings, "", true, nil, nil, newNoopObsrepSender)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestBaseExporterWithOptions(t *testing.T) {
	want := errors.New("my error")
	be, err := newBaseExporter(
		defaultSettings, "", false, nil, nil, newNoopObsrepSender,
		WithStart(func(ctx context.Context, host component.Host) error { return want }),
		WithShutdown(func(ctx context.Context) error { return want }),
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

func TestQueueRetryOptionsWithRequestExporter(t *testing.T) {
	bs, err := newBaseExporter(exportertest.NewNopCreateSettings(), "", true, nil, nil, newNoopObsrepSender,
		WithRetry(configretry.NewDefaultBackOffConfig()))
	require.Nil(t, err)
	require.True(t, bs.requestExporter)
	require.Panics(t, func() {
		_, _ = newBaseExporter(exportertest.NewNopCreateSettings(), "", true, nil, nil, newNoopObsrepSender,
			WithRetry(configretry.NewDefaultBackOffConfig()), WithQueue(NewDefaultQueueSettings()))
	})
}

func TestBaseExporterLogging(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = false
	bs, err := newBaseExporter(set, "", true, nil, nil, newNoopObsrepSender, WithRetry(rCfg))
	require.Nil(t, err)
	require.True(t, bs.requestExporter)
	sendErr := bs.send(context.Background(), newErrorRequest())
	require.Error(t, sendErr)

	require.Len(t, observed.FilterLevelExact(zap.ErrorLevel).All(), 1)
}

func TestStatusReportingOnStart(t *testing.T) {
	for _, tc := range []struct {
		name           string
		statusSettings StatusSettings
		expectedStatus component.Status
		startErr       error
	}{
		{
			name:           "Report status on start enabled / successful startup",
			statusSettings: StatusSettings{ReportOnStart: true},
			expectedStatus: component.StatusOK,
		},
		{
			name:           "Report status on start enabled / startup error",
			statusSettings: StatusSettings{ReportOnStart: true},
			startErr:       assert.AnError,
		},
		{
			name:           "Report status on start disabled / successful startup",
			statusSettings: StatusSettings{ReportOnStart: false},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			createSettings := exportertest.NewNopCreateSettings()
			var lastStatus component.Status
			createSettings.TelemetrySettings.ReportComponentStatus = func(ev *component.StatusEvent) error {
				lastStatus = ev.Status()
				return nil
			}

			be, err := newBaseExporter(
				createSettings, "", false, nil, nil, newNoopObsrepSender,
				WithStart(func(ctx context.Context, host component.Host) error { return tc.startErr }),
				WithStatusReporting(tc.statusSettings),
			)

			require.NoError(t, err)
			require.Equal(t, component.StatusNone, lastStatus)
			require.Equal(t, tc.startErr, be.Start(context.Background(), componenttest.NewNopHost()))
			require.Equal(t, tc.expectedStatus, lastStatus)
		})
	}
}
