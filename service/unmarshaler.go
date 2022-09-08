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

package service // import "go.opentelemetry.io/collector/service"

import (
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/service/internal/configunmarshaler"
	"go.opentelemetry.io/collector/service/telemetry"
)

type configSettings struct {
	Receivers  *configunmarshaler.Receivers  `mapstructure:"receivers"`
	Processors *configunmarshaler.Processors `mapstructure:"processors"`
	Exporters  *configunmarshaler.Exporters  `mapstructure:"exporters"`
	Extensions *configunmarshaler.Extensions `mapstructure:"extensions"`
	Connectors *configunmarshaler.Connectors `mapstructure:"connectors"`
	Service    ConfigService                 `mapstructure:"service"`
}

// unmarshal the configSettings from a confmap.Conf.
// After the config is unmarshalled, `Validate()` must be called to validate.
func unmarshal(v *confmap.Conf, factories component.Factories) (*configSettings, error) {
	// Unmarshal top level sections and validate.
	cfg := &configSettings{
		Receivers:  configunmarshaler.NewReceivers(factories.Receivers),
		Processors: configunmarshaler.NewProcessors(factories.Processors),
		Exporters:  configunmarshaler.NewExporters(factories.Exporters),
		Extensions: configunmarshaler.NewExtensions(factories.Extensions),
		Connectors: configunmarshaler.NewConnectors(factories.Connectors),
		// TODO: Add a component.ServiceFactory to allow this to be defined by the Service.
		Service: ConfigService{
			Telemetry: telemetry.Config{
				Logs: telemetry.LogsConfig{
					Level:       zapcore.InfoLevel,
					Development: false,
					Encoding:    "console",
					Sampling: &telemetry.LogsSamplingConfig{
						Initial:    100,
						Thereafter: 100,
					},
					OutputPaths:       []string{"stderr"},
					ErrorOutputPaths:  []string{"stderr"},
					DisableCaller:     false,
					DisableStacktrace: false,
					InitialFields:     map[string]interface{}(nil),
				},
				Metrics: telemetry.MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: ":8888",
				},
			},
		},
	}

	return cfg, v.Unmarshal(&cfg, confmap.WithErrorUnused())
}
