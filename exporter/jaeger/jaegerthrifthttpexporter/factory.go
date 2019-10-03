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

package jaegerthrifthttpexporter

import (
	"fmt"
	"net/url"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
)

const (
	// The value of "type" key in configuration.
	typeStr = "jaeger_thrift_http"
)

// Factory is the factory for Jaeger Thrift over HTTP exporter.
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
		Timeout: defaultHTTPTimeout,
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *Factory) CreateTraceExporter(
	logger *zap.Logger,
	config configmodels.Exporter,
) (exporter.TraceExporter, error) {

	expCfg := config.(*Config)
	_, err := url.ParseRequestURI(expCfg.URL)
	if err != nil {
		// TODO: Improve error message, see #215
		err = fmt.Errorf(
			"%q config requires a valid \"url\": %v",
			expCfg.Name(),
			err)
		return nil, err
	}

	if expCfg.Timeout <= 0 {
		err := fmt.Errorf(
			"%q config requires a positive value for \"timeout\"",
			expCfg.Name())
		return nil, err
	}

	exp, err := New(
		config,
		expCfg.URL,
		expCfg.Headers,
		expCfg.Timeout)
	if err != nil {
		return nil, err
	}

	return exp, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *Factory) CreateMetricsExporter(
	logger *zap.Logger,
	cfg configmodels.Exporter,
) (exporter.MetricsExporter, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}
