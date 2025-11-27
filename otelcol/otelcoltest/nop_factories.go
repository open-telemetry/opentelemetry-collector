// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest // import "go.opentelemetry.io/collector/otelcol/otelcoltest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service/telemetry"
)

// NopFactories returns a otelcol.Factories with all nop factories.
func NopFactories() (otelcol.Factories, error) {
	var factories otelcol.Factories

	// MakeFactoryMap can never return an error with a single Factory
	factories.Extensions, _ = otelcol.MakeFactoryMap(extensiontest.NewNopFactory())
	factories.Receivers, _ = otelcol.MakeFactoryMap(receivertest.NewNopFactory())
	factories.Exporters, _ = otelcol.MakeFactoryMap(exportertest.NewNopFactory())
	factories.Processors, _ = otelcol.MakeFactoryMap(processortest.NewNopFactory())
	factories.Connectors, _ = otelcol.MakeFactoryMap(connectortest.NewNopFactory())
	factories.Telemetry = nopTelemetryFactory()

	return factories, nil
}

func nopTelemetryFactory() telemetry.Factory {
	return telemetry.NewFactory(
		func() component.Config { return struct{}{} },
	)
}
