// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/internal/componentalias"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/xprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

func TestNewBuildSubCommand(t *testing.T) {
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
		// ensure default providers are referenced by scheme to a module
		ProviderModules: map[string]string{
			"file": "go.opentelemetry.io/collector/confmap/provider/testprovider v1.2.3",
			"env":  "go.opentelemetry.io/collector/confmap/provider/testprovider v1.2.3",
		},
		ConverterModules: []string{
			"go.opentelemetry.io/collector/converter/testconverter v1.2.3",
		},
	}
	cmd := NewCommand(set)
	cmd.SetArgs([]string{"components"})

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "components-output.yaml"))
	require.NoError(t, err)

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	err = cmd.Execute()
	require.NoError(t, err)

	// Trim new line at the end of the two strings to make a better comparison as string() adds an extra new
	// line that makes the test fail.
	assert.Equal(t, strings.TrimSpace(string(expectedOutput)), strings.TrimSpace(b.String()))
}

func TestComponentsStableOutput(t *testing.T) {
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              newNamedNopFactories([]string{"bar", "foo", "baz"}),
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
		// assumes default config provider contains `file`` and `env` schemes`.
		ProviderModules: map[string]string{
			"file": "go.opentelemetry.io/collector/confmap/provider/testprovider v1.2.3",
			"env":  "go.opentelemetry.io/collector/confmap/provider/testprovider v1.2.3",
		},
		ConverterModules: []string{
			"go.opentelemetry.io/collector/converter/baz v1.2.3",
			"go.opentelemetry.io/collector/converter/foo v1.2.3",
			"go.opentelemetry.io/collector/converter/bar v1.2.3",
		},
	}
	cmd := NewCommand(set)
	cmd.SetArgs([]string{"components"})

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "components-output-sorted.yaml"))
	require.NoError(t, err)

	// ensure output is reasonably consistent
	for range 5 {
		b := bytes.NewBufferString("")
		cmd.SetOut(b)
		err = cmd.Execute()
		require.NoError(t, err)
		// Trim new line at the end of the two strings to make a better comparison as string() adds an extra new
		// line that makes the test fail.
		assert.Equal(t, strings.TrimSpace(string(expectedOutput)), strings.TrimSpace(b.String()))
	}
}

func newNamedNopFactories(
	placeholderTypes []string,
) func() (Factories, error) {
	return func() (Factories, error) {
		var factories Factories
		var err error

		if factories.Connectors, err = MakeFactoryMap(newListNamedConnectorNopFactory(
			placeholderTypes,
		)...); err != nil {
			return Factories{}, err
		}
		factories.ConnectorModules = make(map[component.Type]string, len(factories.Connectors))
		for _, con := range factories.Connectors {
			factories.ConnectorModules[con.Type()] = "go.opentelemetry.io/collector/connector/connectortest v1.2.3"
		}

		if factories.Extensions, err = MakeFactoryMap(newListNamedExtensionNopFactory(
			placeholderTypes,
		)...); err != nil {
			return Factories{}, err
		}
		factories.ExtensionModules = make(map[component.Type]string, len(factories.Extensions))
		for _, ext := range factories.Extensions {
			factories.ExtensionModules[ext.Type()] = "go.opentelemetry.io/collector/extension/extensiontest v1.2.3"
		}

		if factories.Receivers, err = MakeFactoryMap(newListNamedReceiverNopFactory(
			placeholderTypes,
		)...); err != nil {
			return Factories{}, err
		}
		factories.ReceiverModules = make(map[component.Type]string, len(factories.Receivers))
		for _, rec := range factories.Receivers {
			factories.ReceiverModules[rec.Type()] = "go.opentelemetry.io/collector/receiver/receivertest v1.2.3"
		}

		if factories.Exporters, err = MakeFactoryMap(newListNamedExporterNopFactory(
			placeholderTypes,
		)...); err != nil {
			return Factories{}, err
		}
		factories.ExporterModules = make(map[component.Type]string, len(factories.Exporters))
		for _, exp := range factories.Exporters {
			factories.ExporterModules[exp.Type()] = "go.opentelemetry.io/collector/exporter/exportertest v1.2.3"
		}

		if factories.Processors, err = MakeFactoryMap(newListNamedProcessorNopFactory(
			placeholderTypes,
		)...); err != nil {
			return Factories{}, err
		}
		factories.ProcessorModules = make(map[component.Type]string, len(factories.Processors))
		for _, proc := range factories.Processors {
			factories.ProcessorModules[proc.Type()] = "go.opentelemetry.io/collector/processor/processortest v1.2.3"
		}

		return factories, nil
	}
}

type nopComponent struct {
	component.StartFunc
	component.ShutdownFunc
}

func newNamedConnecterNopFactory(typeName string) connector.Factory {
	return xconnector.NewFactory(
		component.MustNewType(typeName),
		func() component.Config { return struct{}{} },
	)
}

func newListNamedConnectorNopFactory(typeNames []string) []connector.Factory {
	facts := make([]connector.Factory, 0, len(typeNames))
	for _, typ := range typeNames {
		facts = append(facts, newNamedConnecterNopFactory(typ))
	}
	return facts
}

func newNamedExtensionNopFactory(typeName string) extension.Factory {
	return extension.NewFactory(
		component.MustNewType(typeName),
		func() component.Config { return struct{}{} },
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return nopComponent{}, nil
		},
		component.StabilityLevelStable,
	)
}

func newListNamedExtensionNopFactory(typeNames []string) []extension.Factory {
	facts := make([]extension.Factory, 0, len(typeNames))
	for _, typ := range typeNames {
		facts = append(facts, newNamedExtensionNopFactory(typ))
	}
	return facts
}

func newNamedReceiverNopFactory(typeName string) receiver.Factory {
	return xreceiver.NewFactory(
		component.MustNewType(typeName),
		func() component.Config { return struct{}{} },
	)
}

func newListNamedReceiverNopFactory(typeNames []string) []receiver.Factory {
	facts := make([]receiver.Factory, 0, len(typeNames))
	for _, typ := range typeNames {
		facts = append(facts, newNamedReceiverNopFactory(typ))
	}
	return facts
}

func newNamedProcessorNopFactory(typeName string) processor.Factory {
	return xprocessor.NewFactory(
		component.MustNewType(typeName),
		func() component.Config { return struct{}{} },
	)
}

func newListNamedProcessorNopFactory(typeNames []string) []processor.Factory {
	facts := make([]processor.Factory, 0, len(typeNames))
	for _, typ := range typeNames {
		facts = append(facts, newNamedProcessorNopFactory(typ))
	}
	return facts
}

func newNamedExportersNopFactory(typeName string) exporter.Factory {
	return xexporter.NewFactory(
		component.MustNewType(typeName),
		func() component.Config { return struct{}{} },
	)
}

func newListNamedExporterNopFactory(typeNames []string) []exporter.Factory {
	facts := make([]exporter.Factory, 0, len(typeNames))
	for _, typ := range typeNames {
		facts = append(facts, newNamedExportersNopFactory(typ))
	}
	return facts
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
