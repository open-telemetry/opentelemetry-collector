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
	"go.opentelemetry.io/collector/internal/localhostgate"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
)

const (
	grpcPort = 4317
	httpPort = 4318

	defaultTracesURLPath  = "/v1/traces"
	defaultMetricsURLPath = "/v1/metrics"
	defaultLogsURLPath    = "/v1/logs"
)

// NewFactory creates a new OTLP receiver factory.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithSharedLogs(createLog, metadata.LogsStability, func(r receiver.Logs, l consumer.Logs) {
			r.(*otlpReceiver).registerLogsConsumer(l)
		}),
		receiver.WithSharedMetrics(createMetrics, metadata.MetricsStability, func(r receiver.Metrics, m consumer.Metrics) {
			r.(*otlpReceiver).registerMetricsConsumer(m)
		}),
		receiver.WithSharedTraces(createTraces, metadata.TracesStability, func(r receiver.Traces, t consumer.Traces) {
			r.(*otlpReceiver).registerTraceConsumer(t)
		}),
	)
}

// createDefaultConfig creates the default configuration for receiver.
func createDefaultConfig() component.Config {
	return &Config{
		Protocols: Protocols{
			GRPC: &configgrpc.ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  localhostgate.EndpointForPort(grpcPort),
					Transport: confignet.TransportTypeTCP,
				},
				// We almost write 0 bytes, so no need to tune WriteBufferSize.
				ReadBufferSize: 512 * 1024,
			},
			HTTP: &HTTPConfig{
				ServerConfig: &confighttp.ServerConfig{
					Endpoint: localhostgate.EndpointForPort(httpPort),
				},
				TracesURLPath:  defaultTracesURLPath,
				MetricsURLPath: defaultMetricsURLPath,
				LogsURLPath:    defaultLogsURLPath,
			},
		},
	}
}

// createTraces creates a trace receiver based on provided config.
func createTraces(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	oCfg := cfg.(*Config)
	r, err := newOtlpReceiver(oCfg, &set)
	r.nextTraces = nextConsumer
	return r, err
}

// createMetrics creates a metrics receiver based on provided config.
func createMetrics(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	oCfg := cfg.(*Config)
	r, err := newOtlpReceiver(oCfg, &set)
	r.nextMetrics = nextConsumer
	return r, err
}

// createLog creates a log receiver based on provided config.
func createLog(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	oCfg := cfg.(*Config)
	r, err := newOtlpReceiver(oCfg, &set)
	r.nextLogs = nextConsumer
	return r, err
}
