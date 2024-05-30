// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverexperimental

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/consumerexperimental/consumerexperimentaltest"
	"go.opentelemetry.io/collector/receiver"
)

func TestNewFactory(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg })
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateProfilesReceiver(context.Background(), receiver.CreateSettings{}, &defaultCfg, consumerexperimentaltest.NewNop())
	assert.Error(t, err)
}

func TestNewFactoryWithOptions(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithProfiles(createProfiles, component.StabilityLevelDeprecated))
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDeprecated, factory.ProfilesReceiverStability())
	_, err := factory.CreateProfilesReceiver(context.Background(), receiver.CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func TestMakeFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []Factory
		out  map[component.Type]Factory
	}

	p1 := NewFactory(component.MustNewType("p1"), nil)
	p2 := NewFactory(component.MustNewType("p2"), nil)
	testCases := []testCase{
		{
			name: "different names",
			in:   []Factory{p1, p2},
			out: map[component.Type]Factory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []Factory{p1, p2, NewFactory(component.MustNewType("p1"), nil)},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := MakeFactoryMap([]Factory{
		NewFactory(component.MustNewType("err"), nil),
		NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			WithProfiles(createProfiles, component.StabilityLevelDevelopment),
		),
	}...)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		id           component.ID
		err          string
		nextProfiles consumerexperimental.Profiles
	}{
		{
			name:         "unknown",
			id:           component.MustNewID("unknown"),
			err:          "receiver factory not available for: \"unknown\"",
			nextProfiles: consumerexperimentaltest.NewNop(),
		},
		{
			name:         "err",
			id:           component.MustNewID("err"),
			err:          "telemetry type is not supported",
			nextProfiles: consumerexperimentaltest.NewNop(),
		},
		{
			name:         "all",
			id:           component.MustNewID("all"),
			nextProfiles: consumerexperimentaltest.NewNop(),
		},
		{
			name:         "all/named",
			id:           component.MustNewIDWithName("all", "named"),
			nextProfiles: consumerexperimentaltest.NewNop(),
		},
		{
			name:         "no next consumer",
			id:           component.MustNewID("unknown"),
			err:          "nil next Consumer",
			nextProfiles: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewBuilder(cfgs, factories)

			te, err := b.CreateProfiles(context.Background(), createSettings(tt.id), tt.nextProfiles)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, te)
			}
		})
	}
}

func TestBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := MakeFactoryMap([]Factory{
		NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			WithProfiles(createProfiles, component.StabilityLevelDevelopment),
		),
	}...)

	require.NoError(t, err)

	bErr := NewBuilder(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	te, err := bErr.CreateProfiles(context.Background(), createSettings(missingID), consumerexperimentaltest.NewNop())
	assert.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, te)
}

func TestBuilderFactory(t *testing.T) {
	factories, err := MakeFactoryMap([]Factory{NewFactory(component.MustNewType("foo"), nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := NewBuilder(cfgs, factories)

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
}

var nopInstance = &nopReceiver{
	Consumer: consumerexperimentaltest.NewNop(),
}

// nopReceiver stores consumed profiles for testing purposes.
type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
	consumerexperimentaltest.Consumer
}

func createProfiles(context.Context, receiver.CreateSettings, component.Config, consumerexperimental.Profiles) (Profiles, error) {
	return nopInstance, nil
}

func createSettings(id component.ID) receiver.CreateSettings {
	return receiver.CreateSettings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
