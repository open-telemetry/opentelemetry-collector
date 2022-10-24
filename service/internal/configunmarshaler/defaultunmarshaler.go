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

package configunmarshaler // import "go.opentelemetry.io/collector/service/internal/configunmarshaler"

import (
	"fmt"
	"reflect"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/service/telemetry"
)

// These are errors that can be returned by Unmarshal(). Note that error codes are not part
// of Unmarshal()'s public API, they are for internal unit testing only.
type configErrorCode int

const (
	// Skip 0, start errors codes from 1.
	_ configErrorCode = iota

	errUnmarshalTopLevelStructure
)

type configError struct {
	// the original error.
	error

	// internal error code.
	code configErrorCode
}

type configSettings struct {
	Receivers  *Receivers     `mapstructure:"receivers"`
	Processors *Processors    `mapstructure:"processors"`
	Exporters  *Exporters     `mapstructure:"exporters"`
	Extensions *Extensions    `mapstructure:"extensions"`
	Service    config.Service `mapstructure:"service"`
}

// Unmarshal the config.Config from a confmap.Conf.
// After the config is unmarshalled, `Validate()` must be called to validate.
func Unmarshal(v *confmap.Conf, factories component.Factories) (*config.Config, error) {
	// Unmarshal top level sections and validate.
	rawCfg := configSettings{
		Receivers:  NewReceivers(factories.Receivers),
		Processors: NewProcessors(factories.Processors),
		Exporters:  NewExporters(factories.Exporters),
		Extensions: NewExtensions(factories.Extensions),
		// TODO: Add a component.ServiceFactory to allow this to be defined by the Service.
		Service: config.Service{
			Telemetry: telemetry.Config{
				Logs: telemetry.LogsConfig{
					Level:             zapcore.InfoLevel,
					Development:       false,
					Encoding:          "console",
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
	if err := v.Unmarshal(&rawCfg, confmap.WithErrorUnused()); err != nil {
		return nil, configError{
			error: fmt.Errorf("error reading top level configuration sections: %w", err),
			code:  errUnmarshalTopLevelStructure,
		}
	}

	cfg := &config.Config{
		Receivers:  rawCfg.Receivers.GetReceivers(),
		Processors: rawCfg.Processors.GetProcessors(),
		Exporters:  rawCfg.Exporters.GetExporters(),
		Extensions: rawCfg.Extensions.GetExtensions(),
		Service:    rawCfg.Service,
	}

	return cfg, nil
}

func errorUnknownType(component string, id config.ComponentID, factories []reflect.Value) error {
	return fmt.Errorf("unknown %s type: %q for id: %q (valid values: %v)", component, id.Type(), id, factories)
}

func errorUnmarshalError(component string, id config.ComponentID, err error) error {
	return fmt.Errorf("error reading %s configuration for %q: %w", component, id, err)
}
