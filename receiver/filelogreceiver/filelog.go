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

package filelogreceiver

import (
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/file"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/stanza"
)

const typeStr = "filelog"

// NewFactory creates a factory for filelog receiver
func NewFactory() component.ReceiverFactory {
	return stanza.NewFactory(ReceiverType{})
}

// ReceiverType implements stanza.LogReceiverType
// to create a file tailing receiver
type ReceiverType struct{}

// Type is the receiver type
func (f ReceiverType) Type() config.Type {
	return config.Type(typeStr)
}

// CreateDefaultConfig creates a config with type and version
func (f ReceiverType) CreateDefaultConfig() config.Receiver {
	return createDefaultConfig()
}
func createDefaultConfig() *FileLogConfig {
	return &FileLogConfig{
		BaseConfig: stanza.BaseConfig{
			ReceiverSettings: config.ReceiverSettings{
				TypeVal: config.Type(typeStr),
				NameVal: typeStr,
			},
			Operators: stanza.OperatorConfigs{},
			Converter: stanza.ConverterConfig{
				MaxFlushCount: stanza.DefaultMaxFlushCount,
				FlushInterval: stanza.DefaultFlushInterval,
			},
		},
		Input: stanza.InputConfig{},
	}
}

// BaseConfig gets the base config from config, for now
func (f ReceiverType) BaseConfig(cfg config.Receiver) stanza.BaseConfig {
	return cfg.(*FileLogConfig).BaseConfig
}

// FileLogConfig defines configuration for the filelog receiver
type FileLogConfig struct {
	stanza.BaseConfig `mapstructure:",squash"`
	Input             stanza.InputConfig `mapstructure:",remain"`
}

// DecodeInputConfig unmarshals the input operator
func (f ReceiverType) DecodeInputConfig(cfg config.Receiver) (*operator.Config, error) {
	logConfig := cfg.(*FileLogConfig)
	yamlBytes, _ := yaml.Marshal(logConfig.Input)
	inputCfg := file.NewInputConfig("file_input")
	if err := yaml.Unmarshal(yamlBytes, &inputCfg); err != nil {
		return nil, err
	}
	return &operator.Config{Builder: inputCfg}, nil
}
