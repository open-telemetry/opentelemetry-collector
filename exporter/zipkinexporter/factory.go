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

package zipkinexporter

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "zipkin"

	defaultTimeout = time.Second * 5

	defaultFormat = "json"

	defaultServiceName string = "<missing service name>"
)

// NewFactory creates a factory for Zipkin exporter
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithTraces(createTraceExporter))
}

// CreateDefaultConfig creates the default configuration for exporter.
func createDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Timeout: defaultTimeout,
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
		Format:             defaultFormat,
		DefaultServiceName: defaultServiceName,
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func createTraceExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	config configmodels.Exporter,
) (component.TraceExporter, error) {
	zc := config.(*Config)

	if zc.Endpoint == "" {
		// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/215
		return nil, errors.New("exporter config requires a non-empty 'endpoint'")
	}

	return newTraceExporter(zc)
}
