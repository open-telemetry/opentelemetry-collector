// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampleconnector // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleconnector"

import (
	"context"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleconnector/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// NewFactory returns a connector.Factory for sample connector.
func NewFactory() connector.Factory {
	return connector.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		connector.WithMetricsToMetrics(createMetricsToMetricsConnector, metadata.MetricsToMetricsStability))
}

func createMetricsToMetricsConnector(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Metrics, error) {
	return nopInstance, nil
}

var nopInstance = &nopConnector{}

type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
}

func (n nopConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (n nopConnector) ConsumeMetrics(context.Context, pmetric.Metrics) error {
	return nil
}
