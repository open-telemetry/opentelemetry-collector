// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

const (
	defaultTracesURLPath   = "/v1/traces"
	defaultMetricsURLPath  = "/v1/metrics"
	defaultLogsURLPath     = "/v1/logs"
	defaultProfilesURLPath = "/v1development/profiles"

	transportHTTP = "http"
	transportGRPC = "grpc"
)

// NewFactory creates a new OTLP receiver factory.
func NewFactory() receiver.Factory {
	return xreceiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xreceiver.WithTraces(createTraces, metadata.TracesStability),
		xreceiver.WithMetrics(createMetrics, metadata.MetricsStability),
		xreceiver.WithLogs(createLog, metadata.LogsStability),
		xreceiver.WithProfiles(createProfiles, metadata.ProfilesStability),
	)
}

// createDefaultConfig creates the default configuration for receiver.
func createDefaultConfig() component.Config {
	return &Config{
		Protocols: Protocols{
			GRPC: &configgrpc.ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:4317",
					Transport: confignet.TransportTypeTCP,
				},
				// We almost write 0 bytes, so no need to tune WriteBufferSize.
				ReadBufferSize: 512 * 1024,
			},
			HTTP: &HTTPConfig{
				ServerConfig: &confighttp.ServerConfig{
					Endpoint: "localhost:4318",
				},
				TracesURLPath:  defaultTracesURLPath,
				MetricsURLPath: defaultMetricsURLPath,
				LogsURLPath:    defaultLogsURLPath,
			},
		},
	}
}

// createTraces creates a trace receiver based on provided config.
func createTraces(_ context.Context, set receiver.Settings, cfg component.Config, next consumer.Traces) (receiver.Traces, error) {
	oCfg := cfg.(*Config)
	return newTraces(set, oCfg, next)
}

// createMetrics creates a metrics receiver based on provided config.
func createMetrics(_ context.Context, set receiver.Settings, cfg component.Config, next consumer.Metrics) (receiver.Metrics, error) {
	oCfg := cfg.(*Config)
	return newMetrics(set, oCfg, next)
}

// createLog creates a log receiver based on provided config.
func createLog(_ context.Context, set receiver.Settings, cfg component.Config, next consumer.Logs) (receiver.Logs, error) {
	oCfg := cfg.(*Config)
	return newLogs(set, oCfg, next)
}

// createProfiles creates a trace receiver based on provided config.
func createProfiles(_ context.Context, set receiver.Settings, cfg component.Config, next xconsumer.Profiles) (xreceiver.Profiles, error) {
	oCfg := cfg.(*Config)
	return newProfiles(set, oCfg, next)
}
