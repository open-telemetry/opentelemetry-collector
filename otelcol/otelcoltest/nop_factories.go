// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest // import "go.opentelemetry.io/collector/otelcol/otelcoltest"

import (
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// NopFactories returns a otelcol.Factories with all nop factories.
func NopFactories() (otelcol.Factories, error) {
	var factories otelcol.Factories
	var err error

	if factories.Extensions, err = otelcol.MakeFactoryMap(extensiontest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Receivers, err = otelcol.MakeFactoryMap(receivertest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Exporters, err = otelcol.MakeFactoryMap(exportertest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Processors, err = otelcol.MakeFactoryMap(processortest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Connectors, err = otelcol.MakeFactoryMap(connectortest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	return factories, err
}
