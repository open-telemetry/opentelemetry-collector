// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivertest // import "go.opentelemetry.io/collector/receiver/receivertest"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var defaultComponentType = component.MustNewType("nop")

// NewNopCreateSettings returns a new nop settings for Create*Receiver functions.
func NewNopCreateSettings() receiver.CreateSettings {
	return receiver.CreateSettings{
		ID:                component.NewIDWithName(defaultComponentType, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns a receiver.Factory that constructs nop receivers.
func NewNopFactory(opts ...NopOption) receiver.Factory {
	cfg := defaultNopConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	factoryOpts := make([]receiver.FactoryOption, 0)
	componentType := defaultComponentType
	if cfg.withTraces {
		factoryOpts = append(factoryOpts, receiver.WithTraces(createTraces, component.StabilityLevelStable))
	} else {
		componentType += "_notraces"
	}
	if cfg.withMetrics {
		factoryOpts = append(factoryOpts, receiver.WithMetrics(createMetrics, component.StabilityLevelStable))
	} else {
		componentType += "_nometrics"
	}
	if cfg.withLogs {
		factoryOpts = append(factoryOpts, receiver.WithLogs(createLogs, component.StabilityLevelStable))
	} else {
		componentType += "_nologs"
	}

	return receiver.NewFactory(componentType, func() component.Config { return cfg }, factoryOpts...)
}

type nopConfig struct {
	withTraces  bool
	withMetrics bool
	withLogs    bool
}

func defaultNopConfig() *nopConfig {
	return &nopConfig{
		withTraces:  true,
		withMetrics: true,
		withLogs:    true,
	}
}

type NopOption func(*nopConfig)

// WithoutTraces creates a NopReceiver that cannot produce traces.
func WithoutTraces() NopOption {
	return func(c *nopConfig) {
		c.withTraces = false
	}
}

// WithoutMetrics creates a NopReceiver that cannot produce metrics.
func WithoutMetrics() NopOption {
	return func(c *nopConfig) {
		c.withMetrics = false
	}
}

// WithoutLogs creates a NopReceiver that cannot produce logs.
func WithoutLogs() NopOption {
	return func(c *nopConfig) {
		c.withLogs = false
	}
}

func createTraces(context.Context, receiver.CreateSettings, component.Config, consumer.Traces) (receiver.Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, receiver.CreateSettings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, receiver.CreateSettings, component.Config, consumer.Logs) (receiver.Logs, error) {
	return nopInstance, nil
}

var nopInstance = &nopReceiver{}

// nopReceiver acts as a receiver for testing purposes.
type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
}

// NewNopBuilder returns a receiver.Builder that constructs nop receivers.
func NewNopBuilder() *receiver.Builder {
	nopFactory := NewNopFactory()
	return receiver.NewBuilder(
		map[component.ID]component.Config{component.NewID(defaultComponentType): nopFactory.CreateDefaultConfig()},
		map[component.Type]receiver.Factory{defaultComponentType: nopFactory})
}
