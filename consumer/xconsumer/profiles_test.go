// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconsumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestDefaultProfiles(t *testing.T) {
	cp, err := NewProfiles(func(context.Context, pprofile.Profiles) error { return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, cp.Capabilities())
}

func TestNilFuncProfiles(t *testing.T) {
	_, err := NewProfiles(nil)
	assert.Equal(t, errNilFunc, err)
}

func TestWithCapabilitiesProfiles(t *testing.T) {
	cp, err := NewProfiles(
		func(context.Context, pprofile.Profiles) error { return nil },
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.Equal(t, consumer.Capabilities{MutatesData: true}, cp.Capabilities())
}

func TestConsumeProfiles(t *testing.T) {
	consumeCalled := false
	cp, err := NewProfiles(func(context.Context, pprofile.Profiles) error { consumeCalled = true; return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.True(t, consumeCalled)
}

func TestConsumeProfiles_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	cp, err := NewProfiles(func(context.Context, pprofile.Profiles) error { return want })
	require.NoError(t, err)
	assert.Equal(t, want, cp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}
