// Copyright 2019, OpenTelemetry Authors
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
	"errors"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
)

const (
	// The value of "type" key in configuration.
	typeStr = "zipkin"
)

// Factory is the factory for OpenCensus exporter.
type Factory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *Factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for exporter.
func (f *Factory) CreateDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Format: "json",
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *Factory) CreateTraceExporter(logger *zap.Logger, config configmodels.Exporter) (exporter.TraceExporter, error) {
	cfg := config.(*Config)

	if cfg.URL == "" {
		// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/215
		return nil, errors.New("exporter config requires a non-empty 'url'")
	}
	// <missing service name> is used if the zipkin span is not carrying the name of the service, which shouldn't happen
	// in normal circumstances. It happens only due to (bad) conversions between formats. The current value is a
	// clear indication that somehow the name of the service was lost in translation.
	ze, err := newZipkinExporter(cfg.URL, "<missing service name>", 0, cfg.Format)
	if err != nil {
		return nil, err
	}

	return ze, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *Factory) CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (exporter.MetricsExporter, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}
