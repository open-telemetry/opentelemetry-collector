// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statushelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestWrapConsumeTraces(t *testing.T) {
	for _, tc := range []struct {
		name          string
		retErr        error
		expectedEvent *component.StatusEvent
	}{
		{
			name:          "consume no error",
			expectedEvent: component.NewStatusEvent(component.StatusOK),
		},
		{
			name:          "consume with error",
			retErr:        assert.AnError,
			expectedEvent: component.NewRecoverableErrorEvent(assert.AnError),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			telemetrySettings := componenttest.NewNopTelemetrySettings()
			var lastEvent *component.StatusEvent
			telemetrySettings.ReportComponentStatus = func(ev *component.StatusEvent) error {
				lastEvent = ev
				return nil
			}

			consumeFunc := func(ctx context.Context, td ptrace.Traces) error {
				return tc.retErr
			}

			wrappedConsume := WrapConsumeTraces(consumeFunc, telemetrySettings)

			assert.Equal(t, tc.retErr, wrappedConsume(context.Background(), ptrace.NewTraces()))
			assert.Equal(t, tc.expectedEvent.Status(), lastEvent.Status())
			assert.Equal(t, tc.expectedEvent.Err(), lastEvent.Err())

		})
	}
}

func TestWrapConsumeMetrics(t *testing.T) {
	for _, tc := range []struct {
		name          string
		retErr        error
		expectedEvent *component.StatusEvent
	}{
		{
			name:          "consume no error",
			expectedEvent: component.NewStatusEvent(component.StatusOK),
		},
		{
			name:          "consume with error",
			retErr:        assert.AnError,
			expectedEvent: component.NewRecoverableErrorEvent(assert.AnError),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			telemetrySettings := componenttest.NewNopTelemetrySettings()
			var lastEvent *component.StatusEvent
			telemetrySettings.ReportComponentStatus = func(ev *component.StatusEvent) error {
				lastEvent = ev
				return nil
			}

			consumeFunc := func(ctx context.Context, md pmetric.Metrics) error {
				return tc.retErr
			}

			wrappedConsume := WrapConsumeMetrics(consumeFunc, telemetrySettings)

			assert.Equal(t, tc.retErr, wrappedConsume(context.Background(), pmetric.NewMetrics()))
			assert.Equal(t, tc.expectedEvent.Status(), lastEvent.Status())
			assert.Equal(t, tc.expectedEvent.Err(), lastEvent.Err())

		})
	}
}

func TestWrapConsumeLogs(t *testing.T) {
	for _, tc := range []struct {
		name          string
		retErr        error
		expectedEvent *component.StatusEvent
	}{
		{
			name:          "consume no error",
			expectedEvent: component.NewStatusEvent(component.StatusOK),
		},
		{
			name:          "consume with error",
			retErr:        assert.AnError,
			expectedEvent: component.NewRecoverableErrorEvent(assert.AnError),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			telemetrySettings := componenttest.NewNopTelemetrySettings()
			var lastEvent *component.StatusEvent
			telemetrySettings.ReportComponentStatus = func(ev *component.StatusEvent) error {
				lastEvent = ev
				return nil
			}

			consumeFunc := func(ctx context.Context, ld plog.Logs) error {
				return tc.retErr
			}

			wrappedConsume := WrapConsumeLogs(consumeFunc, telemetrySettings)

			assert.Equal(t, tc.retErr, wrappedConsume(context.Background(), plog.NewLogs()))
			assert.Equal(t, tc.expectedEvent.Status(), lastEvent.Status())
			assert.Equal(t, tc.expectedEvent.Err(), lastEvent.Err())

		})
	}
}
