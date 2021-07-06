// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusremotewriteexporter

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "prometheusremotewrite"
)

// NewFactory creates a new Prometheus Remote Write exporter.
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithMetrics(createMetricsExporter))
}

func createMetricsExporter(_ context.Context, set component.ExporterCreateSettings,
	cfg config.Exporter) (component.MetricsExporter, error) {

	prwCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("invalid configuration")
	}

	prwe, err := NewPRWExporter(prwCfg, set.BuildInfo)
	if err != nil {
		return nil, err
	}

	// Don't allow users to configure the queue.
	// See https://github.com/open-telemetry/opentelemetry-collector/issues/2949.
	// Prometheus remote write samples needs to be in chronological
	// order for each timeseries. If we shard the incoming metrics
	// without considering this limitation, we experience
	// "out of order samples" errors.
	return exporterhelper.NewMetricsExporter(
		cfg,
		set,
		prwe.PushMetrics,
		exporterhelper.WithTimeout(prwCfg.TimeoutSettings),
		exporterhelper.WithQueue(exporterhelper.QueueSettings{
			Enabled:      true,
			NumConsumers: 1,
			QueueSize:    prwCfg.RemoteWriteQueue.QueueSize,
		}),
		exporterhelper.WithRetry(prwCfg.RetrySettings),
		exporterhelper.WithResourceToTelemetryConversion(prwCfg.ResourceToTelemetrySettings),
		exporterhelper.WithStart(prwe.Start),
		exporterhelper.WithShutdown(prwe.Shutdown),
	)
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
		Namespace:        "",
		ExternalLabels:   map[string]string{},
		TimeoutSettings:  exporterhelper.DefaultTimeoutSettings(),
		RetrySettings: exporterhelper.RetrySettings{
			Enabled:         true,
			InitialInterval: 50 * time.Millisecond,
			MaxInterval:     200 * time.Millisecond,
			MaxElapsedTime:  1 * time.Minute,
		},
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "http://some.url:9411/api/prom/push",
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			ReadBufferSize:  0,
			WriteBufferSize: 512 * 1024,
			Timeout:         exporterhelper.DefaultTimeoutSettings().Timeout,
			Headers:         map[string]string{},
		},
		// TODO(jbd): Adjust the default queue size.
		RemoteWriteQueue: RemoteWriteQueue{
			QueueSize:    10000,
			NumConsumers: 5,
		},
	}
}
