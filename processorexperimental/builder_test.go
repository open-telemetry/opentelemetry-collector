package processorexperimental

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/consumerexperimental/consumerexperimentaltest"
	"go.opentelemetry.io/collector/processor"
)

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
			err:          "processor factory not available for: \"unknown\"",
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
	assert.EqualError(t, err, "processor \"all/missing\" is not configured")
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

func createSettings(id component.ID) processor.CreateSettings {
	return processor.CreateSettings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
