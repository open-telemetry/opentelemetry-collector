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

package exporterhelper

import (
	"context"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configerror"
)

// FactoryOption apply changes to ExporterOptions.
type FactoryOption func(o *factory)

// CreateDefaultConfig is the equivalent of component.ExporterFactory.CreateDefaultConfig()
type CreateDefaultConfig func() config.Exporter

// CreateTracesExporter is the equivalent of component.ExporterFactory.CreateTracesExporter()
type CreateTracesExporter func(context.Context, component.ExporterCreateParams, config.Exporter) (component.TracesExporter, error)

// CreateMetricsExporter is the equivalent of component.ExporterFactory.CreateMetricsExporter()
type CreateMetricsExporter func(context.Context, component.ExporterCreateParams, config.Exporter) (component.MetricsExporter, error)

// CreateLogsExporter is the equivalent of component.ExporterFactory.CreateLogsExporter()
type CreateLogsExporter func(context.Context, component.ExporterCreateParams, config.Exporter) (component.LogsExporter, error)

type factory struct {
	cfgType               config.Type
	customUnmarshaler     component.CustomUnmarshaler
	createDefaultConfig   CreateDefaultConfig
	createTracesExporter  CreateTracesExporter
	createMetricsExporter CreateMetricsExporter
	createLogsExporter    CreateLogsExporter
}

// WithTraces overrides the default "error not supported" implementation for CreateTracesReceiver.
func WithTraces(createTraceExporter CreateTracesExporter) FactoryOption {
	return func(o *factory) {
		o.createTracesExporter = createTraceExporter
	}
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsReceiver.
func WithMetrics(createMetricsExporter CreateMetricsExporter) FactoryOption {
	return func(o *factory) {
		o.createMetricsExporter = createMetricsExporter
	}
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsReceiver.
func WithLogs(createLogsExporter CreateLogsExporter) FactoryOption {
	return func(o *factory) {
		o.createLogsExporter = createLogsExporter
	}
}

// WithCustomUnmarshaler implements component.ConfigUnmarshaler.
func WithCustomUnmarshaler(customUnmarshaler component.CustomUnmarshaler) FactoryOption {
	return func(o *factory) {
		o.customUnmarshaler = customUnmarshaler
	}
}

// NewFactory returns a component.ExporterFactory.
func NewFactory(
	cfgType config.Type,
	createDefaultConfig CreateDefaultConfig,
	options ...FactoryOption) component.ExporterFactory {
	f := &factory{
		cfgType:             cfgType,
		createDefaultConfig: createDefaultConfig,
	}
	for _, opt := range options {
		opt(f)
	}
	var ret component.ExporterFactory
	if f.customUnmarshaler != nil {
		ret = &factoryWithUnmarshaler{f}
	} else {
		ret = f
	}
	return ret
}

// Type gets the type of the Exporter config created by this factory.
func (f *factory) Type() config.Type {
	return f.cfgType
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *factory) CreateDefaultConfig() config.Exporter {
	return f.createDefaultConfig()
}

// CreateTracesExporter creates a component.TracesExporter based on this config.
func (f *factory) CreateTracesExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg config.Exporter) (component.TracesExporter, error) {
	if f.createTracesExporter != nil {
		return f.createTracesExporter(ctx, params, cfg)
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsExporter creates a component.MetricsExporter based on this config.
func (f *factory) CreateMetricsExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg config.Exporter) (component.MetricsExporter, error) {
	if f.createMetricsExporter != nil {
		return f.createMetricsExporter(ctx, params, cfg)
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateLogsExporter creates a metrics processor based on this config.
func (f *factory) CreateLogsExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg config.Exporter,
) (component.LogsExporter, error) {
	if f.createLogsExporter != nil {
		return f.createLogsExporter(ctx, params, cfg)
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}

var _ component.ConfigUnmarshaler = (*factoryWithUnmarshaler)(nil)

type factoryWithUnmarshaler struct {
	*factory
}

// Unmarshal un-marshals the config using the provided custom unmarshaler.
func (f *factoryWithUnmarshaler) Unmarshal(componentViperSection *viper.Viper, intoCfg interface{}) error {
	return f.customUnmarshaler(componentViperSection, intoCfg)
}
