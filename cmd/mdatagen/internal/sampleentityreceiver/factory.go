// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampleentityreceiver // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleentityreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleentityreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// NewFactory returns a receiver.Factory for sample entity receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		receiver.WithMetrics(createMetrics, metadata.MetricsStability),
	)
}

func createMetrics(context.Context, receiver.Settings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
	return nopInstance, nil
}

var nopInstance = &nopReceiver{}

type nopReceiver struct {
	component.StartFunc
}

func (nopReceiver) Shutdown(context.Context) error {
	return nil
}
