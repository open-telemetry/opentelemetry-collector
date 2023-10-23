// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testProfilesCfg = struct{}{}

func TestNewProfilesProcessor(t *testing.T) {
	lp, err := NewProfilesProcessor(context.Background(), processortest.NewNopCreateSettings(), &testProfilesCfg, consumertest.NewNop(), newTestLProcessor(nil))
	require.NoError(t, err)

	assert.True(t, lp.Capabilities().MutatesData)
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.NoError(t, lp.Shutdown(context.Background()))
}

func TestNewProfilesProcessor_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	lp, err := NewProfilesProcessor(context.Background(), processortest.NewNopCreateSettings(), &testProfilesCfg, consumertest.NewNop(), newTestLProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	assert.NoError(t, err)

	assert.Equal(t, want, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, lp.Shutdown(context.Background()))
	assert.False(t, lp.Capabilities().MutatesData)
}

func TestNewProfilesProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewProfilesProcessor(context.Background(), processortest.NewNopCreateSettings(), &testProfilesCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)

	_, err = NewProfilesProcessor(context.Background(), processortest.NewNopCreateSettings(), &testProfilesCfg, nil, newTestLProcessor(nil))
	assert.Equal(t, component.ErrNilNextConsumer, err)
}

func TestNewProfilesProcessor_ProcessProfileError(t *testing.T) {
	want := errors.New("my_error")
	lp, err := NewProfilesProcessor(context.Background(), processortest.NewNopCreateSettings(), &testProfilesCfg, consumertest.NewNop(), newTestLProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, lp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}

func TestNewProfilesProcessor_ProcessProfilesErrSkipProcessingData(t *testing.T) {
	lp, err := NewProfilesProcessor(context.Background(), processortest.NewNopCreateSettings(), &testProfilesCfg, consumertest.NewNop(), newTestLProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.Equal(t, nil, lp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}

func newTestLProcessor(retError error) ProcessProfilesFunc {
	return func(_ context.Context, ld pprofile.Profiles) (pprofile.Profiles, error) {
		return ld, retError
	}
}
