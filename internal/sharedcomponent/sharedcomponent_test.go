// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sharedcomponent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

var id = component.MustNewID("test")

type baseComponent struct {
	component.StartFunc
	component.ShutdownFunc
	telemetry *component.TelemetrySettings
}

func TestNewMap(t *testing.T) {
	comps := NewMap[component.ID, *baseComponent]()
	assert.Len(t, comps.components, 0)
}

func TestNewSharedComponentsCreateError(t *testing.T) {
	comps := NewMap[component.ID, *baseComponent]()
	assert.Len(t, comps.components, 0)
	myErr := errors.New("my error")
	_, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return nil, myErr },
		newNopTelemetrySettings(),
	)
	assert.ErrorIs(t, err, myErr)
	assert.Len(t, comps.components, 0)
}

func TestSharedComponentsLoadOrStore(t *testing.T) {
	nop := &baseComponent{}

	comps := NewMap[component.ID, *baseComponent]()
	got, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return nop, nil },
		newNopTelemetrySettings(),
	)
	require.NoError(t, err)
	assert.Len(t, comps.components, 1)
	assert.Same(t, nop, got.Unwrap())
	gotSecond, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { panic("should not be called") },
		newNopTelemetrySettings(),
	)

	require.NoError(t, err)
	assert.Same(t, got, gotSecond)

	// Shutdown nop will remove
	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Len(t, comps.components, 0)
	gotThird, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return nop, nil },
		newNopTelemetrySettings(),
	)
	require.NoError(t, err)
	assert.NotSame(t, got, gotThird)
}

func TestSharedComponent(t *testing.T) {
	wantErr := errors.New("my error")
	calledStart := 0
	calledStop := 0
	comp := &baseComponent{
		StartFunc: func(context.Context, component.Host) error {
			calledStart++
			return wantErr
		},
		ShutdownFunc: func(context.Context) error {
			calledStop++
			return wantErr
		}}

	comps := NewMap[component.ID, *baseComponent]()
	got, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return comp, nil },
		newNopTelemetrySettings(),
	)
	require.NoError(t, err)
	assert.Equal(t, wantErr, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	// Second time is not called anymore.
	assert.NoError(t, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	// first time, shutdown is called.
	assert.Equal(t, wantErr, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
	// Second time is not called anymore.
	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
}
func TestSharedComponentsReportStatus(t *testing.T) {
	reportedStatuses := make(map[*component.InstanceID][]component.Status)
	newStatusFunc := func() func(*component.StatusEvent) {
		instanceID := &component.InstanceID{}
		return func(ev *component.StatusEvent) {
			if ev.Status() == component.StatusNone {
				return
			}
			reportedStatuses[instanceID] = append(reportedStatuses[instanceID], ev.Status())
		}
	}

	comp := &baseComponent{}
	comps := NewMap[component.ID, *baseComponent]()
	var telemetrySettings *component.TelemetrySettings

	// make a shared component that represents three instances
	for i := 0; i < 3; i++ {
		telemetrySettings = newNopTelemetrySettings()
		telemetrySettings.ReportStatus = newStatusFunc()
		// The initial settings for the shared component need to match the ones passed to the first
		// invocation of LoadOrStore so that underlying telemetry settings reference can be used to
		// wrap ReportStatus for subsequently added "instances".
		if i == 0 {
			comp.telemetry = telemetrySettings
		}
		got, err := comps.LoadOrStore(
			id,
			func() (*baseComponent, error) { return comp, nil },
			telemetrySettings,
		)
		require.NoError(t, err)
		assert.Len(t, comps.components, 1)
		assert.Same(t, comp, got.Unwrap())
	}

	// make sure we don't try to represent a fourth instance if we reuse a telemetrySettings
	_, _ = comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return comp, nil },
		telemetrySettings,
	)

	comp.telemetry.ReportStatus(component.NewStatusEvent(component.StatusStarting))

	comp.telemetry.ReportStatus(component.NewStatusEvent(component.StatusOK))

	// simulate an error
	comp.telemetry.ReportStatus(component.NewStatusEvent(component.StatusNone))

	// stopping
	comp.telemetry.ReportStatus(component.NewStatusEvent(component.StatusStopping))

	// stopped
	comp.telemetry.ReportStatus(component.NewStatusEvent(component.StatusStopped))

	// The shared component represents 3 component instances. Reporting status for the shared
	// component should report status for each of the instances it represents.
	expectedStatuses := []component.Status{
		component.StatusStarting,
		component.StatusOK,
		component.StatusStopping,
		component.StatusStopped,
	}

	require.Equal(t, 3, len(reportedStatuses))

	for _, actualStatuses := range reportedStatuses {
		require.Equal(t, expectedStatuses, actualStatuses)
	}
}

func TestReportStatusOnStartShutdown(t *testing.T) {
	for _, tc := range []struct {
		name             string
		startErr         error
		shutdownErr      error
		expectedStatuses []component.Status
	}{
		{
			name:        "successful start/stop",
			startErr:    nil,
			shutdownErr: nil,
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
		},
		{
			name:        "start error",
			startErr:    assert.AnError,
			shutdownErr: nil,
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusPermanentError,
			},
		},
		{
			name:        "shutdown error",
			shutdownErr: assert.AnError,
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusPermanentError,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reportedStatuses := make(map[*component.InstanceID][]component.Status)
			newStatusFunc := func() func(*component.StatusEvent) {
				instanceID := &component.InstanceID{}
				return func(ev *component.StatusEvent) {
					reportedStatuses[instanceID] = append(reportedStatuses[instanceID], ev.Status())
				}
			}
			base := &baseComponent{}
			if tc.startErr != nil {
				base.StartFunc = func(context.Context, component.Host) error {
					return tc.startErr
				}
			}
			if tc.shutdownErr != nil {
				base.ShutdownFunc = func(context.Context) error {
					return tc.shutdownErr
				}
			}
			comps := NewMap[component.ID, *baseComponent]()
			var comp *Component[*baseComponent]
			var err error
			for i := 0; i < 3; i++ {
				telemetrySettings := newNopTelemetrySettings()
				telemetrySettings.ReportStatus = newStatusFunc()
				if i == 0 {
					base.telemetry = telemetrySettings
				}
				comp, err = comps.LoadOrStore(
					id,
					func() (*baseComponent, error) { return base, nil },
					telemetrySettings,
				)
				require.NoError(t, err)
			}

			err = comp.Start(context.Background(), componenttest.NewNopHost())
			require.Equal(t, tc.startErr, err)

			if tc.startErr == nil {
				comp.telemetry.ReportStatus(component.NewStatusEvent(component.StatusOK))

				err = comp.Shutdown(context.Background())
				require.Equal(t, tc.shutdownErr, err)
			}

			require.Equal(t, 3, len(reportedStatuses))

			for _, actualStatuses := range reportedStatuses {
				require.Equal(t, tc.expectedStatuses, actualStatuses)
			}
		})
	}
}

// newNopTelemetrySettings streamlines getting a pointer to a NopTelemetrySettings
func newNopTelemetrySettings() *component.TelemetrySettings {
	set := componenttest.NewNopTelemetrySettings()
	return &set
}
