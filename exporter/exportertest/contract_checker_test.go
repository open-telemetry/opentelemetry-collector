// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exportertest

import (
	"net"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
)

// NewTestRetrySettings returns the default settings for otlp exporter test.
func NewTestRetrySettings() exporterhelper.RetrySettings {
	return exporterhelper.RetrySettings{
		Enabled: true,
		// interval is short for the test purposes
		InitialInterval:     10 * time.Millisecond,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          1.1,
		MaxInterval:         10 * time.Second,
		MaxElapsedTime:      1 * time.Minute,
	}
}

// TestConsumeContract is an example of testing of the exporter for the contract between the
// exporter and the receiver.
func TestConsumeContractOtlpLogs(t *testing.T) {

	ln, err := net.Listen("tcp", "localhost:0")
	mockReceiver := otlpLogsReceiverOnGRPCServer(ln)
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	t.Cleanup(func() { mockReceiver.srv.GracefulStop() })

	cfg := &otlpexporter.Config{
		TimeoutSettings: exporterhelper.TimeoutSettings{},
		QueueSettings:   exporterhelper.QueueSettings{Enabled: false},
		RetrySettings:   NewTestRetrySettings(),
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: ln.Addr().String(),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			}},
	}
	params := CheckConsumeContractParams{
		T:                    t,
		Factory:              otlpexporter.NewFactory(),
		DataType:             component.DataTypeLogs,
		Config:               cfg,
		NumberOfTestElements: 10,
		MockReceiver:         mockReceiver,
	}

	CheckConsumeContract(params)
}

func TestConsumeContractOtlpTraces(t *testing.T) {

	ln, err := net.Listen("tcp", "localhost:0")
	mockReceiver := otlpTracesReceiverOnGRPCServer(ln)
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	t.Cleanup(func() { mockReceiver.srv.GracefulStop() })

	cfg := &otlpexporter.Config{
		TimeoutSettings: exporterhelper.TimeoutSettings{},
		QueueSettings:   exporterhelper.QueueSettings{Enabled: false},
		RetrySettings:   NewTestRetrySettings(),
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: ln.Addr().String(),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			}},
	}
	params := CheckConsumeContractParams{
		T:                    t,
		Factory:              otlpexporter.NewFactory(),
		DataType:             component.DataTypeTraces,
		Config:               cfg,
		NumberOfTestElements: 10,
		MockReceiver:         mockReceiver,
	}

	CheckConsumeContract(params)
}

func TestConsumeContractOtlpMetrics(t *testing.T) {

	ln, err := net.Listen("tcp", "localhost:0")
	mockReceiver := otlpMetricsReceiverOnGRPCServer(ln)
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	t.Cleanup(func() { mockReceiver.srv.GracefulStop() })

	cfg := &otlpexporter.Config{
		TimeoutSettings: exporterhelper.TimeoutSettings{},
		QueueSettings:   exporterhelper.QueueSettings{Enabled: false},
		RetrySettings:   NewTestRetrySettings(),
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: ln.Addr().String(),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			}},
	}
	params := CheckConsumeContractParams{
		T:                    t,
		Factory:              otlpexporter.NewFactory(),
		DataType:             component.DataTypeMetrics,
		Config:               cfg,
		NumberOfTestElements: 10,
		MockReceiver:         mockReceiver,
	}

	CheckConsumeContract(params)
}
