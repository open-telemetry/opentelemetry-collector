// Copyright 2019, OpenCensus Authors
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

package prometheusreceiver

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/pkg/configmodels"
	"github.com/open-telemetry/opentelemetry-service/pkg/factories"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

// This file implements config V2 for Prometheus receiver.

var _ = factories.RegisterReceiverFactory(&ReceiverFactory{})

const (
	// The value of "type" key in configuration.
	typeStr = "prometheus"
)

// ReceiverFactory is the factory for receiver.
type ReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *ReceiverFactory) Type() string {
	return typeStr
}

// CustomUnmarshaler returns custom unmarshaler for this config.
func (f *ReceiverFactory) CustomUnmarshaler() factories.CustomUnmarshaler {
	return CustomUnmarshalerFunc
}

// CustomUnmarshalerFunc performs custom unmarshaling of config.
func CustomUnmarshalerFunc(v *viper.Viper, viperKey string, intoCfg interface{}) error {
	// We need custom unmarshaling because prometheus "config" subkey defines its own
	// YAML unmarshaling routines so we need to do it explicitly.

	// Unmarshal our config values (using viper's mapstructure)
	err := v.UnmarshalKey(viperKey, intoCfg)
	if err != nil {
		return fmt.Errorf("prometheus receiver failed to parse config: %s", err)
	}

	// Unmarshal prometheus's config values. Since prometheus uses `yaml` tags, so use `yaml`.
	vSub := v.Sub(viperKey)
	if vSub == nil || !vSub.IsSet(prometheusConfigKey) {
		return nil
	}
	promCfgMap := vSub.Sub(prometheusConfigKey).AllSettings()
	out, err := yaml.Marshal(promCfgMap)
	if err != nil {
		return fmt.Errorf("prometheus receiver failed to marshal config to yaml: %s", err)
	}

	config := intoCfg.(*ConfigV2)

	err = yaml.Unmarshal(out, &config.PrometheusConfig)
	if err != nil {
		return fmt.Errorf("prometheus receiver failed to unmarshal yaml to prometheus config: %s", err)
	}
	if len(config.PrometheusConfig.ScrapeConfigs) == 0 {
		return errNilScrapeConfig
	}
	return nil
}

// CreateDefaultConfig creates the default configuration for receiver.
func (f *ReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &ConfigV2{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal:  typeStr,
			NameVal:  typeStr,
			Endpoint: "127.0.0.1:9090",
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *ReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	// Prometheus does not support traces
	return nil, factories.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *ReceiverFactory) CreateMetricsReceiver(
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {

	rCfg := cfg.(*ConfigV2)

	// Create receiver Configuration from our input cfg
	config := Configuration{
		BufferCount:  rCfg.BufferCount,
		BufferPeriod: rCfg.BufferPeriod,
		ScrapeConfig: rCfg.PrometheusConfig,
	}

	if config.ScrapeConfig == nil || len(config.ScrapeConfig.ScrapeConfigs) == 0 {
		return nil, errNilScrapeConfig
	}
	pr := &Preceiver{
		cfg:      &config,
		consumer: consumer,
	}
	return pr, nil
}
