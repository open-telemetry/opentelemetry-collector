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
)

const (
	// The value of "type" key in configuration.
	typeStr = "otlp"
)

// Factory is the factory for OpenCensus exporter.
type Factory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for exporter.
func (f *Factory) CreateDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Headers: map[string]string{},
		},
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *Factory) CreateTraceExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.TraceExporter, error) {
	return NewTraceExporter(ctx, params, cfg)
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *Factory) CreateMetricsExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.MetricsExporter, error) {
	return NewMetricsExporter(ctx, params, cfg)
}

// CreateLogExporter creates a log exporter based on this config.
func (f *Factory) CreateLogExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.LogExporter, error) {
	return NewLogExporter(ctx, params, cfg)
}
