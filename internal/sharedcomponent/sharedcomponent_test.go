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
}

func TestNewSharedComponents(t *testing.T) {
	comps := NewSharedComponents[component.ID, *baseComponent]()
	assert.Len(t, comps.comps, 0)
}

func TestNewSharedComponentsCreateError(t *testing.T) {
	comps := NewSharedComponents[component.ID, *baseComponent]()
	assert.Len(t, comps.comps, 0)
	myErr := errors.New("my error")
	_, err := comps.GetOrAdd(id, func() (*baseComponent, error) { return nil, myErr })
	assert.ErrorIs(t, err, myErr)
	assert.Len(t, comps.comps, 0)
}

func TestSharedComponentsGetOrAdd(t *testing.T) {
	nop := &baseComponent{}

	comps := NewSharedComponents[component.ID, *baseComponent]()
	got, err := comps.GetOrAdd(id, func() (*baseComponent, error) { return nop, nil })
	require.NoError(t, err)
	assert.Len(t, comps.comps, 1)
	assert.Same(t, nop, got.Unwrap())
	gotSecond, err := comps.GetOrAdd(id, func() (*baseComponent, error) { panic("should not be called") })
	require.NoError(t, err)
	assert.Same(t, got, gotSecond)

	// Shutdown nop will remove
	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Len(t, comps.comps, 0)
	gotThird, err := comps.GetOrAdd(id, func() (*baseComponent, error) { return nop, nil })
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
	got, err := comps.GetOrAdd(id, func() (*baseComponent, error) { return comp, nil })
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
