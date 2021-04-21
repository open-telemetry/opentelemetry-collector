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

package stanza

import (
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/config"
)

// BaseConfig is the common configuration of a stanza-based receiver
type BaseConfig struct {
	config.ReceiverSettings `mapstructure:",squash"`
	Operators               OperatorConfigs `mapstructure:"operators"`
	Converter               ConverterConfig `mapstructure:"converter"`
}

// OperatorConfigs is an alias that allows for unmarshaling outside of mapstructure
// Stanza operators should will be migrated to mapstructure for greater compatibility
// but this allows a temporary solution
type OperatorConfigs []map[string]interface{}

// ConverterConfig controls how the internal entry.Entry to pdata.Logs converter
// works.
type ConverterConfig struct {
	// MaxFlushCount defines the maximum number of entries that can be
	// accumulated before flushing them for further processing.
	MaxFlushCount uint `mapstructure:"max_flush_count"`
	// FlushInterval defines how often to flush the converted and accumulated
	// log entries.
	FlushInterval time.Duration `mapstructure:"flush_interval"`
	// WorkerCount defines how many worker goroutines used for entry.Entry to
	// log records translation should be spawned.
	// By default: runtime.NumCPU() / 4 worker are spawned.
	WorkerCount int `mapstructure:"worker_count"`
}

// InputConfig is an alias that allows unmarshaling outside of mapstructure
// This is meant to be used only for the input operator
type InputConfig map[string]interface{}

// decodeOperatorConfigs is an unmarshaling workaround for stanza operators
// This is needed only until stanza operators are migrated to mapstructure
func (cfg BaseConfig) decodeOperatorConfigs() ([]operator.Config, error) {
	if len(cfg.Operators) == 0 {
		return []operator.Config{}, nil
	}

	yamlBytes, _ := yaml.Marshal(cfg.Operators)
	operatorCfgs := []operator.Config{}
	if err := yaml.Unmarshal(yamlBytes, &operatorCfgs); err != nil {
		return nil, err
	}
	return operatorCfgs, nil
}
