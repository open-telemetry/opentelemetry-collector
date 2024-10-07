// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/nopexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func nopFactories() (Factories, error) {
	var factories Factories
	var err error

	if factories.Connectors, err = connector.MakeFactoryMap(connectortest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ConnectorModules = make(map[component.Type]string, len(factories.Connectors))
	for _, con := range factories.Connectors {
		factories.ConnectorModules[con.Type()] = "go.opentelemetry.io/collector/connector/connectortest v1.2.3"
	}

	if factories.Extensions, err = extension.MakeFactoryMap(extensiontest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ExtensionModules = make(map[component.Type]string, len(factories.Extensions))
	for _, ext := range factories.Extensions {
		factories.ExtensionModules[ext.Type()] = "go.opentelemetry.io/collector/extension/extensiontest v1.2.3"
	}

	if factories.Receivers, err = receiver.MakeFactoryMap(receivertest.NewNopFactory(), receivertest.NewNopFactoryForType(pipeline.SignalLogs)); err != nil {
		return Factories{}, err
	}
	factories.ReceiverModules = make(map[component.Type]string, len(factories.Receivers))
	for _, rec := range factories.Receivers {
		factories.ReceiverModules[rec.Type()] = "go.opentelemetry.io/collector/receiver/receivertest v1.2.3"
	}

	if factories.Exporters, err = exporter.MakeFactoryMap(nopexporter.NewFactory()); err != nil {
		return Factories{}, err
	}
	factories.ExporterModules = make(map[component.Type]string, len(factories.Exporters))
	for _, exp := range factories.Exporters {
		factories.ExporterModules[exp.Type()] = "go.opentelemetry.io/collector/exporter/exportertest v1.2.3"
	}

	if factories.Processors, err = processor.MakeFactoryMap(processortest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ProcessorModules = make(map[component.Type]string, len(factories.Processors))
	for _, proc := range factories.Processors {
		factories.ProcessorModules[proc.Type()] = "go.opentelemetry.io/collector/processor/processortest v1.2.3"
	}

	return factories, err
}
