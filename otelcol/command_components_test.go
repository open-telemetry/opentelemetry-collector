// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/componentalias"
)

func TestNewBuildSubCommand(t *testing.T) {
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
		ProviderModules: map[string]string{
			"nop": "go.opentelemetry.io/collector/confmap/provider/testprovider v1.2.3",
		},
		ConverterModules: []string{
			"go.opentelemetry.io/collector/converter/testconverter v1.2.3",
		},
	}
	cmd := NewCommand(set)
	cmd.SetArgs([]string{"components"})

	ExpectedOutput, err := os.ReadFile(filepath.Join("testdata", "components-output.yaml"))
	require.NoError(t, err)

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	err = cmd.Execute()
	require.NoError(t, err)

	// Trim new line at the end of the two strings to make a better comparison as string() adds an extra new
	// line that makes the test fail.
	assert.Equal(t, strings.ReplaceAll(strings.ReplaceAll(string(ExpectedOutput), "\n", ""), "\r", ""), strings.ReplaceAll(strings.ReplaceAll(b.String(), "\n", ""), "\r", ""))
}

type mockFactory struct {
	componentalias.TypeAliasHolder
	name string
}

func (mockFactory) CreateDefaultConfig() component.Config {
	return nil
}

func (m mockFactory) Type() component.Type {
	return component.MustNewType(m.name)
}

func newMockFactory(name string) mockFactory {
	return mockFactory{
		TypeAliasHolder: componentalias.NewTypeAliasHolder(),
		name:            name,
	}
}

func TestSortFactoriesByType(t *testing.T) {
	for _, tt := range []struct {
		name      string
		factories map[component.Type]mockFactory
		want      []mockFactory
	}{
		{
			name:      "with an empty map",
			factories: map[component.Type]mockFactory{},
			want:      []mockFactory{},
		},
		{
			name: "with a single factory",
			factories: map[component.Type]mockFactory{
				component.MustNewType("receiver"): newMockFactory("receiver_factory"),
			},
			want: []mockFactory{
				newMockFactory("receiver_factory"),
			},
		},
		{
			name: "with multiple factories",
			factories: map[component.Type]mockFactory{
				component.MustNewType("processor"): newMockFactory("processor_factory"),
				component.MustNewType("exporter"):  newMockFactory("exporter_factory"),
				component.MustNewType("receiver"):  newMockFactory("receiver_factory"),
			},
			want: []mockFactory{
				newMockFactory("exporter_factory"),
				newMockFactory("processor_factory"),
				newMockFactory("receiver_factory"),
			},
		},
		{
			name: "with aliases factories",
			factories: func() map[component.Type]mockFactory {
				alias := newMockFactory("alias_processor_factory")
				alias.SetDeprecatedAlias(alias.Type())

				return map[component.Type]mockFactory{
					component.MustNewType("processor"):       newMockFactory("processor_factory"),
					component.MustNewType("alias_processor"): alias,
				}
			}(),
			want: []mockFactory{
				newMockFactory("processor_factory"),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := sortFactoriesByType(tt.factories)
			assert.Equal(t, tt.want, got)
		})
	}
}
