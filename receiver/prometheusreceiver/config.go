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

package prometheusreceiver

import (
	"time"

	"github.com/prometheus/prometheus/config"

	"go.opentelemetry.io/collector/config/configmodels"
)

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
