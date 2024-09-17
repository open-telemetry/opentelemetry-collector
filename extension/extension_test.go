// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

type nopExtension struct {
	component.StartFunc
	component.ShutdownFunc
	Settings
}

func TestNewFactory(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	nopExtensionInstance := new(nopExtension)

	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		func(context.Context, Settings, component.Config) (Extension, error) {
			return nopExtensionInstance, nil
		},
		component.StabilityLevelDevelopment)
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, factory.ExtensionStability())
	ext, err := factory.CreateExtension(context.Background(), Settings{}, &defaultCfg)
	require.NoError(t, err)
	assert.Same(t, nopExtensionInstance, ext)
}

func TestMakeFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []Factory
		out  map[component.Type]Factory
	}

	p1 := NewFactory(component.MustNewType("p1"), nil, nil, component.StabilityLevelAlpha)
	p2 := NewFactory(component.MustNewType("p2"), nil, nil, component.StabilityLevelAlpha)
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
			in:   []Factory{p1, p2, NewFactory(component.MustNewType("p1"), nil, nil, component.StabilityLevelAlpha)},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}
