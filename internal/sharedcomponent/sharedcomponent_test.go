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
	// Cached error is returned on subsequent calls to start.
	assert.Equal(t, wantErr, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	// first time, shutdown is called.
	assert.Equal(t, wantErr, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
	// Second time is not called anymore.
	require.NoError(t, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
}

func TestReportStatusRoutedToAllInstances(t *testing.T) {
	// The wrapper reports no lifecycle status of its own. It routes the wrapped
	// component's own status reports to every instance and replays them to instances
	// that register after the component has started.
	reportedStatuses := make(map[*componentstatus.InstanceID][]componentstatus.Status)
	newStatusFunc := func(id *componentstatus.InstanceID, ev *componentstatus.Event) {
		reportedStatuses[id] = append(reportedStatuses[id], ev.Status())
	}

	// The wrapped component reports a runtime status during Start.
	base := &baseComponent{
		StartFunc: func(_ context.Context, host component.Host) error {
			componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusRecoverableError))
			return nil
		},
	}

	comps := NewMap[component.ID, *baseComponent]()
	baseHost := componenttest.NewNopHost()

	// Three pipeline instances share the component. The first Start actually starts it;
	// the other two register afterwards and must still observe the reported status.
	var comp *Component[*baseComponent]
	for range 3 {
		var err error
		comp, err = comps.LoadOrStore(id, func() (*baseComponent, error) { return base, nil })
		require.NoError(t, err)
		require.NoError(t, comp.Start(context.Background(), &testHost{Host: baseHost, InstanceID: &componentstatus.InstanceID{}, newStatusFunc: newStatusFunc}))
	}

	// Every instance observed the RecoverableError and no wrapper-emitted lifecycle events.
	require.Len(t, reportedStatuses, 3)
	for _, statuses := range reportedStatuses {
		assert.Equal(t, []componentstatus.Status{componentstatus.StatusRecoverableError}, statuses)
	}

	// Shutdown does not emit any status from the wrapper either.
	require.NoError(t, comp.Shutdown(context.Background()))
	require.Len(t, reportedStatuses, 3)
	for _, statuses := range reportedStatuses {
		assert.Equal(t, []componentstatus.Status{componentstatus.StatusRecoverableError}, statuses)
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
