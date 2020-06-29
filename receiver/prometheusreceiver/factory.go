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

package prometheusreceiver

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

// This file implements config for Prometheus receiver.

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
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CustomUnmarshaler returns custom unmarshaler for this config.
func (f *Factory) CustomUnmarshaler() component.CustomUnmarshaler {
	return CustomUnmarshalerFunc
}

// CustomUnmarshalerFunc performs custom unmarshaling of config.
func CustomUnmarshalerFunc(componentViperSection *viper.Viper, intoCfg interface{}) error {
	if componentViperSection == nil {
		return nil
	}
	// We need custom unmarshaling because prometheus "config" subkey defines its own
	// YAML unmarshaling routines so we need to do it explicitly.

	err := componentViperSection.UnmarshalExact(intoCfg)
	if err != nil {
		return fmt.Errorf("prometheus receiver failed to parse config: %s", err)
	}

	// Unmarshal prometheus's config values. Since prometheus uses `yaml` tags, so use `yaml`.
	if !componentViperSection.IsSet(prometheusConfigKey) {
		return nil
	}
	promCfgMap := componentViperSection.Sub(prometheusConfigKey).AllSettings()
	out, err := yaml.Marshal(promCfgMap)
	if err != nil {
		return fmt.Errorf("prometheus receiver failed to marshal config to yaml: %s", err)
	}
	config := intoCfg.(*Config)

	err = yaml.UnmarshalStrict(out, &config.PrometheusConfig)
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
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *Factory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumerOld,
) (component.TraceReceiver, error) {
	// Prometheus does not support traces
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *Factory) CreateMetricsReceiver(ctx context.Context, logger *zap.Logger, cfg configmodels.Receiver, nextConsumer consumer.MetricsConsumerOld) (component.MetricsReceiver, error) {
	config := cfg.(*Config)
	if config.PrometheusConfig == nil || len(config.PrometheusConfig.ScrapeConfigs) == 0 {
		return nil, errNilScrapeConfig
	}
	return newPrometheusReceiver(logger, config, nextConsumer), nil
}
