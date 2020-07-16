// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlpexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "otlp"
)

// NewFactory creates a factory for OTLP exporter.
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithTraces(createTraceExporter),
		exporterhelper.WithMetrics(createMetricsExporter),
		exporterhelper.WithLogs(createLogExporter))
}

func createDefaultConfig() configmodels.Exporter {
	// TODO: Enable the queued settings.
	qs := exporterhelper.CreateDefaultQueuedSettings()
	qs.Disabled = true
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		TimeoutSettings: exporterhelper.CreateDefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.CreateDefaultRetrySettings(),
		QueuedSettings:  qs,
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Headers: map[string]string{},
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
}

func createTraceExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.TraceExporter, error) {
	oce, err := newExporter(cfg)
	if err != nil {
		return nil, err
	}
	oCfg := cfg.(*Config)
	oexp, err := exporterhelper.NewTraceExporter(
		cfg,
		oce.pushTraceData,
		exporterhelper.WithTimeout(oCfg.TimeoutSettings),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueued(oCfg.QueuedSettings),
		exporterhelper.WithShutdown(oce.shutdown))
	if err != nil {
		return nil, err
	}

	return oexp, nil
}

func createMetricsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.MetricsExporter, error) {
	oce, err := newExporter(cfg)
	if err != nil {
		return nil, err
	}
	oCfg := cfg.(*Config)
	oexp, err := exporterhelper.NewMetricsExporter(
		cfg,
		oce.pushMetricsData,
		exporterhelper.WithTimeout(oCfg.TimeoutSettings),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueued(oCfg.QueuedSettings),
		exporterhelper.WithShutdown(oce.shutdown),
	)
	if err != nil {
		return nil, err
	}

	return oexp, nil
}

func createLogExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.LogExporter, error) {
	oce, err := newExporter(cfg)
	if err != nil {
		return nil, err
	}
	oCfg := cfg.(*Config)
	oexp, err := exporterhelper.NewLogsExporter(
		cfg,
		oce.pushLogData,
		exporterhelper.WithTimeout(oCfg.TimeoutSettings),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueued(oCfg.QueuedSettings),
		exporterhelper.WithShutdown(oce.shutdown),
	)
	if err != nil {
		return nil, err
	}

	return oexp, nil
}
