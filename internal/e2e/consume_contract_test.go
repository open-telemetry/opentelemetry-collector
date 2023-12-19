// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

func testExporterConfig(endpoint string) component.Config {
	retryConfig := configretry.NewDefaultBackOffConfig()
	retryConfig.InitialInterval = time.Millisecond // interval is short for the test purposes
	return &otlpexporter.Config{
		QueueConfig: exporterhelper.QueueSettings{Enabled: false},
		RetryConfig: retryConfig,
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: endpoint,
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		},
	}
}

func testReceiverConfig(endpoint string) component.Config {
	cfg := otlpreceiver.NewFactory().CreateDefaultConfig()
	cfg.(*otlpreceiver.Config).HTTP = nil
	cfg.(*otlpreceiver.Config).GRPC.NetAddr.Endpoint = endpoint
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
		DataType:             component.DataTypeLogs,
		ExporterConfig:       testExporterConfig(addr),
		ReceiverFactory:      otlpreceiver.NewFactory(),
		ReceiverConfig:       testReceiverConfig(addr),
	})
}

func TestConsumeContractOtlpTraces(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	exportertest.CheckConsumeContract(exportertest.CheckConsumeContractParams{
		T:                    t,
		NumberOfTestElements: 10,
		DataType:             component.DataTypeTraces,
		ExporterFactory:      otlpexporter.NewFactory(),
		ExporterConfig:       testExporterConfig(addr),
		ReceiverFactory:      otlpreceiver.NewFactory(),
		ReceiverConfig:       testReceiverConfig(addr),
	})
}

func TestConsumeContractOtlpMetrics(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	exportertest.CheckConsumeContract(exportertest.CheckConsumeContractParams{
		T:                    t,
		NumberOfTestElements: 10,
		ExporterFactory:      otlpexporter.NewFactory(),
		DataType:             component.DataTypeMetrics,
		ExporterConfig:       testExporterConfig(addr),
		ReceiverFactory:      otlpreceiver.NewFactory(),
		ReceiverConfig:       testReceiverConfig(addr),
	})
}
