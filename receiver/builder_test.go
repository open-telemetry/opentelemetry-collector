// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := MakeFactoryMap([]Factory{
		NewFactory(component.MustNewType("err"), nil),
		NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			WithTraces(createTraces, component.StabilityLevelDevelopment),
			WithMetrics(createMetrics, component.StabilityLevelAlpha),
			WithLogs(createLogs, component.StabilityLevelDeprecated),
		),
	}...)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		id          component.ID
		err         string
		nextTraces  consumer.Traces
		nextLogs    consumer.Logs
		nextMetrics consumer.Metrics
	}{
		{
			name:        "unknown",
			id:          component.MustNewID("unknown"),
			err:         "receiver factory not available for: \"unknown\"",
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),
		},
		{
			name:        "err",
			id:          component.MustNewID("err"),
			err:         "telemetry type is not supported",
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),
		},
		{
			name:        "all",
			id:          component.MustNewID("all"),
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),
		},
		{
			name:        "all/named",
			id:          component.MustNewIDWithName("all", "named"),
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),
		},
		{
			name:        "no next consumer",
			id:          component.MustNewID("unknown"),
			err:         "nil next Consumer",
			nextTraces:  nil,
			nextLogs:    nil,
			nextMetrics: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewBuilder(cfgs, factories)

			te, err := b.CreateTraces(context.Background(), createSettings(tt.id), tt.nextTraces)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, te)
			}

			me, err := b.CreateMetrics(context.Background(), createSettings(tt.id), tt.nextMetrics)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, me)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, me)
			}

			le, err := b.CreateLogs(context.Background(), createSettings(tt.id), tt.nextLogs)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, le)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, le)
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
			WithTraces(createTraces, component.StabilityLevelDevelopment),
			WithMetrics(createMetrics, component.StabilityLevelAlpha),
			WithLogs(createLogs, component.StabilityLevelDeprecated),
		),
	}...)

	require.NoError(t, err)

	bErr := NewBuilder(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	te, err := bErr.CreateTraces(context.Background(), createSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, te)

	me, err := bErr.CreateMetrics(context.Background(), createSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, me)

	le, err := bErr.CreateLogs(context.Background(), createSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, le)
}

func TestBuilderFactory(t *testing.T) {
	factories, err := MakeFactoryMap([]Factory{NewFactory(component.MustNewType("foo"), nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := NewBuilder(cfgs, factories)

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
}
