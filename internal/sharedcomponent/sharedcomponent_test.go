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
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
)

var id = component.MustNewID("test")

type baseComponent struct {
	component.StartFunc
	component.ShutdownFunc
}

func TestNewMap(t *testing.T) {
	comps := NewMap[component.ID, *baseComponent]()
	assert.Empty(t, comps.components)
}

func TestNewSharedComponentsCreateError(t *testing.T) {
	comps := NewMap[component.ID, *baseComponent]()
	assert.Empty(t, comps.components)
	myErr := errors.New("my error")
	_, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return nil, myErr },
	)
	require.ErrorIs(t, err, myErr)
	assert.Empty(t, comps.components)
}

func TestSharedComponentsLoadOrStore(t *testing.T) {
	nop := &baseComponent{}

	comps := NewMap[component.ID, *baseComponent]()
	got, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return nop, nil },
	)
	require.NoError(t, err)
	assert.Len(t, comps.components, 1)
	assert.Same(t, nop, got.Unwrap())
	gotSecond, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { panic("should not be called") },
	)

	require.NoError(t, err)
	assert.Same(t, got, gotSecond)

	// Shutdown nop will remove
	require.NoError(t, got.Shutdown(context.Background()))
	assert.Empty(t, comps.components)
	gotThird, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return nop, nil },
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
		},
	}

	comps := NewMap[component.ID, *baseComponent]()
	got, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return comp, nil },
	)
	require.NoError(t, err)
	assert.Equal(t, wantErr, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	// Second time is not called anymore.
	require.NoError(t, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	// first time, shutdown is called.
	assert.Equal(t, wantErr, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
	// Second time is not called anymore.
	require.NoError(t, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
}

func TestReportStatusOnStartShutdown(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		startErr                     error
		shutdownErr                  error
		expectedStatuses             []componentstatus.Status
		expectedNumReporterInstances int
	}{
		{
			name:        "successful start/stop",
			startErr:    nil,
			shutdownErr: nil,
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
			expectedNumReporterInstances: 3,
		},
		{
			name:        "start error",
			startErr:    assert.AnError,
			shutdownErr: nil,
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusPermanentError,
			},
			expectedNumReporterInstances: 1,
		},
		{
			name:        "shutdown error",
			shutdownErr: assert.AnError,
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusPermanentError,
			},
			expectedNumReporterInstances: 3,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reportedStatuses := make(map[*componentstatus.InstanceID][]componentstatus.Status)
			newStatusFunc := func(id *componentstatus.InstanceID, ev *componentstatus.Event) {
				reportedStatuses[id] = append(reportedStatuses[id], ev.Status())
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
			for range 3 {
				comp, err = comps.LoadOrStore(
					id,
					func() (*baseComponent, error) { return base, nil },
				)
				require.NoError(t, err)
			}

			baseHost := componenttest.NewNopHost()
			for range 3 {
				err = comp.Start(context.Background(), &testHost{Host: baseHost, InstanceID: &componentstatus.InstanceID{}, newStatusFunc: newStatusFunc})
				if err != nil {
					break
				}
			}

			require.Equal(t, tc.startErr, err)

			if tc.startErr == nil {
				comp.hostWrapper.Report(componentstatus.NewEvent(componentstatus.StatusOK))

				err = comp.Shutdown(context.Background())
				require.Equal(t, tc.shutdownErr, err)
			}

			require.Len(t, reportedStatuses, tc.expectedNumReporterInstances)

			for _, actualStatuses := range reportedStatuses {
				require.Equal(t, tc.expectedStatuses, actualStatuses)
			}
		})
	}
}

var (
	_ component.Host           = (*testHost)(nil)
	_ componentstatus.Reporter = (*testHost)(nil)
)

type testHost struct {
	component.Host
	*componentstatus.InstanceID
	newStatusFunc func(id *componentstatus.InstanceID, ev *componentstatus.Event)
}

func (h *testHost) Report(e *componentstatus.Event) {
	h.newStatusFunc(h.InstanceID, e)
}
