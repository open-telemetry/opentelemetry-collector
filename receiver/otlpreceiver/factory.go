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
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receiverprofiles"
)

const (
	defaultTracesURLPath   = "/v1/traces"
	defaultMetricsURLPath  = "/v1/metrics"
	defaultLogsURLPath     = "/v1/logs"
	defaultProfilesURLPath = "/v1development/profiles"
)

// NewFactory creates a new OTLP receiver factory.
func NewFactory() receiver.Factory {
	return receiverprofiles.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiverprofiles.WithTraces(createTraces, metadata.TracesStability),
		receiverprofiles.WithMetrics(createMetrics, metadata.MetricsStability),
		receiverprofiles.WithLogs(createLog, metadata.LogsStability),
		receiverprofiles.WithProfiles(createProfiles, metadata.ProfilesStability),
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
func createTraces(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	oCfg := cfg.(*Config)
	r, err := receivers.LoadOrStore(
		oCfg,
		func() (*otlpReceiver, error) {
			return newOtlpReceiver(oCfg, &set)
		},
		&set.TelemetrySettings,
	)
	if err != nil {
		return nil, err
	}

	r.Unwrap().registerTraceConsumer(nextConsumer)
	return r, nil
}

// createMetrics creates a metrics receiver based on provided config.
func createMetrics(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	oCfg := cfg.(*Config)
	r, err := receivers.LoadOrStore(
		oCfg,
		func() (*otlpReceiver, error) {
			return newOtlpReceiver(oCfg, &set)
		},
		&set.TelemetrySettings,
	)
	if err != nil {
		return nil, err
	}

	r.Unwrap().registerMetricsConsumer(consumer)
	return r, nil
}

// createLog creates a log receiver based on provided config.
func createLog(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	oCfg := cfg.(*Config)
	r, err := receivers.LoadOrStore(
		oCfg,
		func() (*otlpReceiver, error) {
			return newOtlpReceiver(oCfg, &set)
		},
		&set.TelemetrySettings,
	)
	if err != nil {
		return nil, err
	}

	r.Unwrap().registerLogsConsumer(consumer)
	return r, nil
}

// createProfiles creates a trace receiver based on provided config.
func createProfiles(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumerprofiles.Profiles,
) (receiverprofiles.Profiles, error) {
	oCfg := cfg.(*Config)
	r, err := receivers.LoadOrStore(
		oCfg,
		func() (*otlpReceiver, error) {
			return newOtlpReceiver(oCfg, &set)
		},
		&set.TelemetrySettings,
	)
	if err != nil {
		return nil, err
	}

	r.Unwrap().registerProfilesConsumer(nextConsumer)
	return r, nil
}

// This is the map of already created OTLP receivers for particular configurations.
// We maintain this map because the receiver.Factory is asked trace and metric receivers separately
// when it gets CreateTraces() and CreateMetrics() but they must not
// create separate objects, they must use one otlpReceiver object per configuration.
// When the receiver is shutdown it should be removed from this map so the same configuration
// can be recreated successfully.
var receivers = sharedcomponent.NewMap[*Config, *otlpReceiver]()
