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

package prometheusreceiver

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
)

// This file implements config V2 for Prometheus receiver.

var _ (receiver.Factory) = (*Factory)(nil)

const (
	// The value of "type" key in configuration.
	typeStr = "prometheus"

	// The key for Prometheus scraping configs.
	prometheusConfigKey = "config"
)

var (
	errNilScrapeConfig = errors.New("expecting a non-nil ScrapeConfig")
)

// Factory is the factory for receiver.
type Factory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *Factory) Type() string {
	return typeStr
}

// CustomUnmarshaler returns custom unmarshaler for this config.
func (f *Factory) CustomUnmarshaler() receiver.CustomUnmarshaler {
	return CustomUnmarshalerFunc
}

// CustomUnmarshalerFunc performs custom unmarshaling of config.
func CustomUnmarshalerFunc(v *viper.Viper, viperKey string, sourceViperSection *viper.Viper, intoCfg interface{}) error {
	// We need custom unmarshaling because prometheus "config" subkey defines its own
	// YAML unmarshaling routines so we need to do it explicitly.

	// Unmarshal our config values (using viper's mapstructure)
	errorOnUnused := func(decoderCfg *mapstructure.DecoderConfig) {
		// If ErrorUnused is true, then it is an error for there to exist
		// keys in the original map that were unused in the decoding process
		// (extra keys).
		decoderCfg.ErrorUnused = true
	}
	err := v.UnmarshalKey(viperKey, intoCfg, errorOnUnused)
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
	out = []byte(os.ExpandEnv(string(out)))
	config := intoCfg.(*Config)

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
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal:  typeStr,
			NameVal:  typeStr,
			Endpoint: "localhost:9090",
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *Factory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	// Prometheus does not support traces
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *Factory) CreateMetricsReceiver(
	logger *zap.Logger,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	config := cfg.(*Config)
	if config.PrometheusConfig == nil || len(config.PrometheusConfig.ScrapeConfigs) == 0 {
		return nil, errNilScrapeConfig
	}
	return newPrometheusReceiver(logger, config, consumer), nil
}
