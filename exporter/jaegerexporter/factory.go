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

package jaegerexporter

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "jaeger"
)

// NewFactory creates a factory for Jaeger exporter
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithTraces(createTracesExporter))
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
		TimeoutSettings:  exporterhelper.DefaultTimeoutSettings(),
		RetrySettings:    exporterhelper.DefaultRetrySettings(),
		QueueSettings:    exporterhelper.DefaultQueueSettings(),
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
}

func createTracesExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	config config.Exporter,
) (component.TracesExporter, error) {

	expCfg := config.(*Config)
	if expCfg.Endpoint == "" {
		// TODO: Improve error message, see #215
		return nil, fmt.Errorf(
			"%q config requires a non-empty \"endpoint\"",
			expCfg.ID().String())
	}

	return newTracesExporter(expCfg, set)
}
