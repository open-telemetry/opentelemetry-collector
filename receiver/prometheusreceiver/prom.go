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

// Package prometheusreceiver has the logic for scraping Prometheus metrics from
// already instrumented applications and then passing them onto a metricsink instance.
package prometheusreceiver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/prometheus/config"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"

	_ "github.com/prometheus/prometheus/discovery/install" // init() of this package registers various Prometheus service discovery mechanism.
)

const (
	// The value of "type" key in configuration.
	typeStr = "prometheus"

	// The key for Prometheus scraping configs.
	prometheusConfigKey = "config"
)

var errNilScrapeConfig = errors.New("expecting a non-nil ScrapeConfig")

// Config defines configuration for Prometheus receiver.
type Config struct {
	configmodels.ReceiverSettings `mapstructure:",squash"`
	PrometheusConfig              *config.Config `mapstructure:"-"`
	BufferPeriod                  time.Duration  `mapstructure:"buffer_period"`
	BufferCount                   int            `mapstructure:"buffer_count"`
	UseStartTimeMetric            bool           `mapstructure:"use_start_time_metric"`
	StartTimeMetricRegex          string         `mapstructure:"start_time_metric_regex"`

	// ConfigPlaceholder is just an entry to make the configuration pass a check
	// that requires that all keys present in the config actually exist on the
	// structure, ie.: it will error if an unknown key is present.
	ConfigPlaceholder interface{} `mapstructure:"config"`
}

func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithMetrics(createMetricsReceiver),
		receiverhelper.WithCustomUnmarshaler(customUnmarshaler))
}

func customUnmarshaler(componentViperSection *viper.Viper, intoCfg interface{}) error {
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

func createDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

func createMetricsReceiver(
	_ context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	config := cfg.(*Config)
	if config.PrometheusConfig == nil || len(config.PrometheusConfig.ScrapeConfigs) == 0 {
		return nil, errNilScrapeConfig
	}
	return newPrometheusReceiver(params.Logger, config, nextConsumer), nil
}
