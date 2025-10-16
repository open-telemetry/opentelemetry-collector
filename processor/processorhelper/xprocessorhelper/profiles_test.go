// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xprocessorhelper

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testProfilesCfg = struct{}{}

func TestNewProfiles(t *testing.T) {
	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(nil))
	require.NoError(t, err)

	assert.True(t, pp.Capabilities().MutatesData)
	assert.NoError(t, pp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, pp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.NoError(t, pp.Shutdown(context.Background()))
}

func TestNewProfiles_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	require.NoError(t, err)

	assert.Equal(t, want, pp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, pp.Shutdown(context.Background()))
	assert.False(t, pp.Capabilities().MutatesData)
}

func TestNewProfiles_NilRequiredFields(t *testing.T) {
	_, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)
}

func TestNewProfiles_ProcessProfileError(t *testing.T) {
	want := errors.New("my_error")
	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, pp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}

func TestNewProfiles_ProcessProfilesErrSkipProcessingData(t *testing.T) {
	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(processorhelper.ErrSkipProcessingData))
	require.NoError(t, err)
	assert.NoError(t, pp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}

func newTestPProcessor(retError error) ProcessProfilesFunc {
	return func(_ context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
		return pd, retError
	}
}

func TestProfilesConcurrency(t *testing.T) {
	profilesFunc := func(_ context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
		return pd, nil
	}

	incomingProfiles := pprofile.NewProfiles()
	ps := incomingProfiles.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()

	// Add 3 profiles to the incoming
	ps.AppendEmpty()
	ps.AppendEmpty()
	ps.AppendEmpty()

	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), profilesFunc)
	require.NoError(t, err)
	assert.NoError(t, pp.Start(context.Background(), componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10000 {
				assert.NoError(t, pp.ConsumeProfiles(context.Background(), incomingProfiles))
			}
		}()
	}
	wg.Wait()
	assert.NoError(t, pp.Shutdown(context.Background()))
}
