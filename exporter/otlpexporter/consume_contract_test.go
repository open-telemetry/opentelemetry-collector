// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter

import (
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/collector/config/confignet"

	"github.com/cenkalti/backoff/v4"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

// newTestRetrySettings returns the default settings for otlp exporter test.
func newTestRetrySettings() exporterhelper.RetrySettings {
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

func testConfig() component.Config {
	return &Config{
		TimeoutSettings: exporterhelper.TimeoutSettings{},
		QueueSettings:   exporterhelper.QueueSettings{Enabled: false},
		RetrySettings:   newTestRetrySettings(),
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: confignet.NetAddr{Endpoint: fmt.Sprintf("0.0.0.0:4317"), Transport: "tcp"}.Endpoint,
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			}},
	}
}

// Define a function that matches the MockReceiverFactory signature
func createMockOtlpReceiver(decisionFunc exportertest.DecisionFunc) exportertest.MockReceiver {
	mockConsumer := exportertest.CreateDefaultConsumer(decisionFunc)
	rcv := newOTLPDataReceiver(&mockConsumer)
	err := rcv.Start()
	if err != nil {
		return nil
	}
	return rcv
}

// TestConsumeContract is an example of testing of the exporter for the contract between the
// exporter and the receiver.
func TestConsumeContractOtlpLogs(t *testing.T) {

	params := exportertest.CheckConsumeContractParams{
		T:                    t,
		Factory:              NewFactory(),
		DataType:             component.DataTypeLogs,
		Config:               testConfig(),
		NumberOfTestElements: 10,
		MockReceiverFactory:  createMockOtlpReceiver,
	}

	exportertest.CheckConsumeContract(params)
}

func TestConsumeContractOtlpTraces(t *testing.T) {

	params := exportertest.CheckConsumeContractParams{
		T:                    t,
		Factory:              NewFactory(),
		DataType:             component.DataTypeTraces,
		Config:               testConfig(),
		NumberOfTestElements: 10,
		MockReceiverFactory:  createMockOtlpReceiver,
	}

	exportertest.CheckConsumeContract(params)
}

func TestConsumeContractOtlpMetrics(t *testing.T) {

	params := exportertest.CheckConsumeContractParams{
		T:                    t,
		Factory:              NewFactory(),
		DataType:             component.DataTypeMetrics,
		Config:               testConfig(),
		NumberOfTestElements: 10,
		MockReceiverFactory:  createMockOtlpReceiver,
	}

	exportertest.CheckConsumeContract(params)
}
