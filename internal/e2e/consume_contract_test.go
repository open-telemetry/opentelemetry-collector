// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// GetOrInsertDefault is a helper function to get or insert a default value for a configoptional.Optional type.
func GetOrInsertDefault[T any](t *testing.T, opt *configoptional.Optional[T]) *T {
	if opt.HasValue() {
		return opt.Get()
	}

	empty := confmap.NewFromStringMap(map[string]any{})
	require.NoError(t, empty.Unmarshal(opt))
	val := opt.Get()
	require.NotNil(t, "Expected a default value to be set for %T", val)
	return val
}

func testExporterConfig(endpoint string) component.Config {
	retryConfig := configretry.NewDefaultBackOffConfig()
	retryConfig.InitialInterval = time.Millisecond // interval is short for the test purposes
	return &otlpexporter.Config{
		QueueConfig: exporterhelper.QueueBatchConfig{Enabled: false},
		RetryConfig: retryConfig,
		ClientConfig: configgrpc.ClientConfig{
			Endpoint: endpoint,
			TLS: configtls.ClientConfig{
				Insecure: true,
			},
		},
	}
}

func testReceiverConfig(t *testing.T, endpoint string) component.Config {
	cfg := otlpreceiver.NewFactory().CreateDefaultConfig()
	GetOrInsertDefault(t, &cfg.(*otlpreceiver.Config).GRPC).NetAddr.Endpoint = endpoint
	return cfg
}

// TestConsumeContract is an example of testing of the exporter for the contract between the
// exporter and the receiver.
func TestConsumeContractOtlpLogs(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	exportertest.CheckConsumeContract(exportertest.CheckConsumeContractParams{
		T:                    t,
		NumberOfTestElements: 10,
		ExporterFactory:      otlpexporter.NewFactory(),
		Signal:               pipeline.SignalLogs,
		ExporterConfig:       testExporterConfig(addr),
		ReceiverFactory:      otlpreceiver.NewFactory(),
		ReceiverConfig:       testReceiverConfig(t, addr),
	})
}

func TestConsumeContractOtlpTraces(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	exportertest.CheckConsumeContract(exportertest.CheckConsumeContractParams{
		T:                    t,
		NumberOfTestElements: 10,
		Signal:               pipeline.SignalTraces,
		ExporterFactory:      otlpexporter.NewFactory(),
		ExporterConfig:       testExporterConfig(addr),
		ReceiverFactory:      otlpreceiver.NewFactory(),
		ReceiverConfig:       testReceiverConfig(t, addr),
	})
}

func TestConsumeContractOtlpMetrics(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	exportertest.CheckConsumeContract(exportertest.CheckConsumeContractParams{
		T:                    t,
		NumberOfTestElements: 10,
		ExporterFactory:      otlpexporter.NewFactory(),
		Signal:               pipeline.SignalMetrics,
		ExporterConfig:       testExporterConfig(addr),
		ReceiverFactory:      otlpreceiver.NewFactory(),
		ReceiverConfig:       testReceiverConfig(t, addr),
	})
}
