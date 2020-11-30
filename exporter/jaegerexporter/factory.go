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
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
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
		exporterhelper.WithTraces(createTraceExporter))
}

func createDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		TimeoutSettings: exporterhelper.DefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.DefaultRetrySettings(),
		QueueSettings:   exporterhelper.DefaultQueueSettings(),
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
}

func createTraceExporter(
	_ context.Context,
	params component.ExporterCreateParams,
	config configmodels.Exporter,
) (component.TracesExporter, error) {

	expCfg := config.(*Config)
	if expCfg.Endpoint == "" {
		// TODO: Improve error message, see #215
		err := fmt.Errorf(
			"%q config requires a non-empty \"endpoint\"",
			expCfg.Name())
		return nil, err
	}

	exp, err := newTraceExporter(expCfg, params.Logger)
	if err != nil {
		return nil, err
	}

	return exp, nil
}
