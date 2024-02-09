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
}

func TestNewMap(t *testing.T) {
	comps := NewMap[component.ID, *baseComponent](true)
	assert.Len(t, comps.components, 0)
}

func TestNewSharedComponentsCreateError(t *testing.T) {
	comps := NewMap[component.ID, *baseComponent](true)
	assert.Len(t, comps.components, 0)
	myErr := errors.New("my error")
	_, _, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return nil, myErr },
	)
	assert.ErrorIs(t, err, myErr)
	assert.Len(t, comps.components, 0)
}

func TestSharedComponentsLoadOrStore(t *testing.T) {
	nop := &baseComponent{}

	comps := NewMap[component.ID, *baseComponent](true)
	got, comp, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return nop, nil },
	)
	require.NoError(t, err)
	assert.Len(t, comps.components, 1)
	assert.Same(t, nop, comp)
	gotSecond, comp2, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { panic("should not be called") },
	)

	require.NoError(t, err)
	assert.Same(t, got, gotSecond)
	assert.Same(t, comp, comp2)

	assert.NoError(t, got.Start(context.Background(), componenttest.NewNopHost()))

	// Shutdown nop will remove
	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Len(t, comps.components, 0)
	gotThird, _, err := comps.LoadOrStore(
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
		StartFunc: func(ctx context.Context, host component.Host) error {
			calledStart++
			return wantErr
		},
		ShutdownFunc: func(ctx context.Context) error {
			calledStop++
			return wantErr
		}}

	comps := NewMap[component.ID, *baseComponent](true)
	got, _, err := comps.LoadOrStore(
		id,
		func() (*baseComponent, error) { return comp, nil },
	)
	require.NoError(t, err)
	assert.Equal(t, wantErr, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	// Second time is not called anymore.
	assert.Equal(t, wantErr, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	// first time, shutdown is not called.
	assert.Equal(t, nil, got.Shutdown(context.Background()))
	assert.Equal(t, 0, calledStop)
	// Second time shutdown is called.
	assert.Equal(t, wantErr, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
}
