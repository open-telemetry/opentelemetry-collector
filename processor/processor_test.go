// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestNewFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(
		typeStr,
		func() component.Config { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesProcessor(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsProcessor(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsProcessor(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewFactoryWithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(
		typeStr,
		func() component.Config { return &defaultCfg },
		WithTraces(createTraces, component.StabilityLevelAlpha),
		WithMetrics(createMetrics, component.StabilityLevelBeta),
		WithLogs(createLogs, component.StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesProcessorStability())
	_, err := factory.CreateTracesProcessor(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsProcessorStability())
	_, err = factory.CreateMetricsProcessor(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsProcessorStability())
	_, err = factory.CreateLogsProcessor(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func TestMakeFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []Factory
		out  map[component.Type]Factory
	}

	p1 := NewFactory("p1", nil)
	p2 := NewFactory("p2", nil)
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
			in:   []Factory{p1, p2, NewFactory("p1", nil)},
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
		NewFactory("err", nil),
		NewFactory(
			"all",
			func() component.Config { return &defaultCfg },
			WithTraces(createTraces, component.StabilityLevelDevelopment),
			WithMetrics(createMetrics, component.StabilityLevelAlpha),
			WithLogs(createLogs, component.StabilityLevelDeprecated),
		),
	}...)
	require.NoError(t, err)

	testCases := []struct {
		name string
		id   component.ID
		err  string
	}{
		{
			name: "unknown",
			id:   component.NewID("unknown"),
			err:  "processor factory not available for: \"unknown\"",
		},
		{
			name: "err",
			id:   component.NewID("err"),
			err:  "telemetry type is not supported",
		},
		{
			name: "all",
			id:   component.NewID("all"),
		},
		{
			name: "all/named",
			id:   component.NewIDWithName("all", "named"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewBuilder(cfgs, factories)

			te, err := b.CreateTraces(context.Background(), createSettings(tt.id), nil)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, te)
			}

			me, err := b.CreateMetrics(context.Background(), createSettings(tt.id), nil)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, me)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, me)
			}

			le, err := b.CreateLogs(context.Background(), createSettings(tt.id), nil)
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
			"all",
			func() component.Config { return &defaultCfg },
			WithTraces(createTraces, component.StabilityLevelDevelopment),
			WithMetrics(createMetrics, component.StabilityLevelAlpha),
			WithLogs(createLogs, component.StabilityLevelDeprecated),
		),
	}...)

	require.NoError(t, err)

	bErr := NewBuilder(map[component.ID]component.Config{}, factories)
	missingID := component.NewIDWithName("all", "missing")

	te, err := bErr.CreateTraces(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "processor \"all/missing\" is not configured")
	assert.Nil(t, te)

	me, err := bErr.CreateMetrics(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "processor \"all/missing\" is not configured")
	assert.Nil(t, me)

	le, err := bErr.CreateLogs(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "processor \"all/missing\" is not configured")
	assert.Nil(t, le)
}

func TestBuilderGetters(t *testing.T) {
	factories, err := MakeFactoryMap([]Factory{NewFactory("foo", nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.NewID("foo"): struct{}{}}
	b := NewBuilder(cfgs, factories)

	assert.NotNil(t, b.Config(component.NewID("foo")))
	assert.Nil(t, b.Config(component.NewID("bar")))

	assert.NotNil(t, b.Factory(component.NewID("foo").Type()))
	assert.Nil(t, b.Factory(component.NewID("bar").Type()))
}

var nopInstance = &nopProcessor{
	Consumer: consumertest.NewNop(),
}

// nopProcessor stores consumed traces and metrics for testing purposes.
type nopProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createTraces(context.Context, CreateSettings, component.Config, consumer.Traces) (Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, CreateSettings, component.Config, consumer.Metrics) (Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, CreateSettings, component.Config, consumer.Logs) (Logs, error) {
	return nopInstance, nil
}

func createSettings(id component.ID) CreateSettings {
	return CreateSettings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
