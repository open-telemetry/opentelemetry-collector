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

var id = component.NewID("test")

type baseComponent struct {
	component.StartFunc
	component.ShutdownFunc
	telemetry *component.TelemetrySettings
}

func TestNewSharedComponents(t *testing.T) {
	comps := NewSharedComponents[component.ID, *baseComponent]()
	assert.Len(t, comps.comps, 0)
}

func TestNewSharedComponentsCreateError(t *testing.T) {
	comps := NewSharedComponents[component.ID, *baseComponent]()
	assert.Len(t, comps.comps, 0)
	myErr := errors.New("my error")
	_, err := comps.GetOrAdd(
		id,
		func() (*baseComponent, error) { return nil, myErr },
		newNopTelemetrySettings(),
	)
	assert.ErrorIs(t, err, myErr)
	assert.Len(t, comps.comps, 0)
}

func TestSharedComponentsGetOrAdd(t *testing.T) {
	nop := &baseComponent{}

	comps := NewSharedComponents[component.ID, *baseComponent]()
	got, err := comps.GetOrAdd(
		id,
		func() (*baseComponent, error) { return nop, nil },
		newNopTelemetrySettings(),
	)
	require.NoError(t, err)
	assert.Len(t, comps.comps, 1)
	assert.Same(t, nop, got.Unwrap())
	gotSecond, err := comps.GetOrAdd(
		id,
		func() (*baseComponent, error) { panic("should not be called") },
		newNopTelemetrySettings(),
	)

	require.NoError(t, err)
	assert.Same(t, got, gotSecond)

	// Shutdown nop will remove
	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Len(t, comps.comps, 0)
	gotThird, err := comps.GetOrAdd(
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
		StartFunc: func(ctx context.Context, host component.Host) error {
			calledStart++
			return wantErr
		},
		ShutdownFunc: func(ctx context.Context) error {
			calledStop++
			return wantErr
		}}

	comps := NewSharedComponents[component.ID, *baseComponent]()
	got, err := comps.GetOrAdd(
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
	assert.Equal(t, wantErr, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
	// Second time is not called anymore.
	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
}

func TestSharedComponentsReportStatus(t *testing.T) {
	reportedStatuses := make(map[*component.InstanceID][]component.Status)
	newStatusFunc := func() func(*component.StatusEvent) error {
		instanceID := &component.InstanceID{}
		return func(ev *component.StatusEvent) error {
			// Use an event with component.StatusNone to simulate an error.
			if ev.Status() == component.StatusNone {
				return assert.AnError
			}
			reportedStatuses[instanceID] = append(reportedStatuses[instanceID], ev.Status())
			return nil
		}
	}

	comp := &baseComponent{}
	comps := NewSharedComponents[component.ID, *baseComponent]()
	var telemetrySettings *component.TelemetrySettings

	// make a shared component that represents three instances
	for i := 0; i < 3; i++ {
		telemetrySettings = newNopTelemetrySettings()
		telemetrySettings.ReportComponentStatus = newStatusFunc()
		// The initial settings for the shared component need to match the ones passed to the first
		// invocation of GetOrAdd so that underlying telemetry settings reference can be used to
		// wrap ReportComponentStatus for subsequently added "instances".
		if i == 0 {
			comp.telemetry = telemetrySettings
		}
		got, err := comps.GetOrAdd(
			id,
			func() (*baseComponent, error) { return comp, nil },
			telemetrySettings,
		)
		require.NoError(t, err)
		assert.Len(t, comps.comps, 1)
		assert.Same(t, comp, got.Unwrap())
	}

	// make sure we don't try to represent a fourth instance if we reuse a telemetrySettings
	_, _ = comps.GetOrAdd(
		id,
		func() (*baseComponent, error) { return comp, nil },
		telemetrySettings,
	)

	err := comp.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStarting))
	require.NoError(t, err)

	// ok
	err = comp.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusOK))
	require.NoError(t, err)

	// simulate an error
	err = comp.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusNone))
	require.ErrorIs(t, err, assert.AnError)

	// stopping
	err = comp.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStopping))
	require.NoError(t, err)

	// stopped
	err = comp.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStopped))
	require.NoError(t, err)

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
			newStatusFunc := func() func(*component.StatusEvent) error {
				instanceID := &component.InstanceID{}
				return func(ev *component.StatusEvent) error {
					reportedStatuses[instanceID] = append(reportedStatuses[instanceID], ev.Status())
					return nil
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
			comps := NewSharedComponents[component.ID, *baseComponent]()
			var comp *SharedComponent[*baseComponent]
			var err error
			for i := 0; i < 3; i++ {
				telemetrySettings := newNopTelemetrySettings()
				telemetrySettings.ReportComponentStatus = newStatusFunc()
				if i == 0 {
					base.telemetry = telemetrySettings
				}
				comp, err = comps.GetOrAdd(
					id,
					func() (*baseComponent, error) { return base, nil },
					telemetrySettings,
				)
				require.NoError(t, err)
			}

			err = comp.Start(context.Background(), componenttest.NewNopHost())
			require.Equal(t, tc.startErr, err)

			if tc.startErr == nil {
				err = comp.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusOK))
				require.NoError(t, err)

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
