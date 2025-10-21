// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service/telemetry"
)

func nopFactories() (Factories, error) {
	var factories Factories
	var err error

	if factories.Connectors, err = MakeFactoryMap(connectortest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ConnectorModules = make(map[component.Type]string, len(factories.Connectors))
	for _, con := range factories.Connectors {
		factories.ConnectorModules[con.Type()] = "go.opentelemetry.io/collector/connector/connectortest v1.2.3"
	}

	if factories.Extensions, err = MakeFactoryMap(extensiontest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ExtensionModules = make(map[component.Type]string, len(factories.Extensions))
	for _, ext := range factories.Extensions {
		factories.ExtensionModules[ext.Type()] = "go.opentelemetry.io/collector/extension/extensiontest v1.2.3"
	}

	if factories.Receivers, err = MakeFactoryMap(receivertest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ReceiverModules = make(map[component.Type]string, len(factories.Receivers))
	for _, rec := range factories.Receivers {
		factories.ReceiverModules[rec.Type()] = "go.opentelemetry.io/collector/receiver/receivertest v1.2.3"
	}

	if factories.Exporters, err = MakeFactoryMap(exportertest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ExporterModules = make(map[component.Type]string, len(factories.Exporters))
	for _, exp := range factories.Exporters {
		factories.ExporterModules[exp.Type()] = "go.opentelemetry.io/collector/exporter/exportertest v1.2.3"
	}

	if factories.Processors, err = MakeFactoryMap(processortest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ProcessorModules = make(map[component.Type]string, len(factories.Processors))
	for _, proc := range factories.Processors {
		factories.ProcessorModules[proc.Type()] = "go.opentelemetry.io/collector/processor/processortest v1.2.3"
	}

	factories.Telemetry = telemetry.NewFactory(func() component.Config {
		return fakeTelemetryConfig{}
	})

	return factories, err
}

func TestMakeFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []component.Factory
		out  map[component.Type]component.Factory
	}

	fRec := receiver.NewFactory(component.MustNewType("rec"), nil)
	fRec2 := receiver.NewFactory(component.MustNewType("rec"), nil)
	fPro := processor.NewFactory(component.MustNewType("pro"), nil)
	fCon := connector.NewFactory(component.MustNewType("con"), nil)
	fExp := exporter.NewFactory(component.MustNewType("exp"), nil)
	fExt := extension.NewFactory(component.MustNewType("ext"), nil, nil, component.StabilityLevelUndefined)
	testCases := []testCase{
		{
			name: "different names",
			in:   []component.Factory{fRec, fPro, fCon, fExp, fExt},
			out: map[component.Type]component.Factory{
				fRec.Type(): fRec,
				fPro.Type(): fPro,
				fCon.Type(): fCon,
				fExp.Type(): fExp,
				fExt.Type(): fExt,
			},
		},
		{
			name: "same name",
			in:   []component.Factory{fRec, fPro, fCon, fExp, fExt, fRec2},
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
