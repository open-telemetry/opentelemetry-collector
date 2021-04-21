// Copyright 2021 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package syslogreceiver

import (
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/syslog"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/tcp"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/udp"
	syslogparser "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/syslog"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/stanza"
)

const typeStr = "syslog"

// NewFactory creates a factory for filelog receiver
func NewFactory() component.ReceiverFactory {
	return stanza.NewFactory(ReceiverType{})
}

// ReceiverType implements stanza.LogReceiverType
// to create a syslog receiver
type ReceiverType struct{}

// Type is the receiver type
func (f ReceiverType) Type() config.Type {
	return config.Type(typeStr)
}

// CreateDefaultConfig creates a config with type and version
func (f ReceiverType) CreateDefaultConfig() config.Receiver {
	return &SysLogConfig{
		BaseConfig: stanza.BaseConfig{
			ReceiverSettings: config.ReceiverSettings{
				TypeVal: config.Type(typeStr),
				NameVal: typeStr,
			},
			Operators: stanza.OperatorConfigs{},
		},
		Input: stanza.InputConfig{},
	}
}

// BaseConfig gets the base config from config, for now
func (f ReceiverType) BaseConfig(cfg config.Receiver) stanza.BaseConfig {
	return cfg.(*SysLogConfig).BaseConfig
}

// SysLogConfig defines configuration for the syslog receiver
type SysLogConfig struct {
	stanza.BaseConfig `mapstructure:",squash"`
	Input             stanza.InputConfig `mapstructure:",remain"`
}

// DecodeInputConfig unmarshals the input operator
func (f ReceiverType) DecodeInputConfig(cfg config.Receiver) (*operator.Config, error) {
	logConfig := cfg.(*SysLogConfig)
	yamlBytes, _ := yaml.Marshal(logConfig.Input)
	inputCfg := syslog.NewSyslogInputConfig("syslog_input")
	inputCfg.SyslogParserConfig = *syslogparser.NewSyslogParserConfig("syslog_parser")

	if err := yaml.Unmarshal(yamlBytes, &inputCfg); err != nil {
		return nil, err
	}

	//
	if inputCfg.Tcp != nil {
		inputCfg.Tcp.InputConfig = tcp.NewTCPInputConfig("tcp_input").InputConfig
	}
	if inputCfg.Udp != nil {
		inputCfg.Udp.InputConfig = udp.NewUDPInputConfig("udp_input").InputConfig
	}

	return &operator.Config{Builder: inputCfg}, nil
}
