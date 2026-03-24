// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampleconnector // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleconnector"

import (
	"context"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleconnector/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// NewFactory returns a connector.Factory for sample connector.
func NewFactory() connector.Factory {
	return xconnector.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		xconnector.WithMetricsToMetrics(createMetricsToMetricsConnector, metadata.MetricsToMetricsStability),
		xconnector.WithProfilesToProfiles(createProfilesToProfilesConnector, metadata.ProfilesToProfilesStability),
	)
}

func createMetricsToMetricsConnector(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Metrics, error) {
	return nopInstance, nil
}

func createProfilesToProfilesConnector(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (xconnector.Profiles, error) {
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

func (n nopConnector) ConsumeProfiles(context.Context, pprofile.Profiles) error {
	return nil
}
