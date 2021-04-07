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

package prometheusexporter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"go.opentelemetry.io/collector/config"
)

// Config defines configuration for Prometheus exporter.
type Config struct {
	*config.ExporterSettings `mapstructure:"-"`

	// The address on which the Prometheus scrape handler will be run on.
	Endpoint string `mapstructure:"endpoint"`

	// Namespace if set, exports metrics under the provided value.
	Namespace string `mapstructure:"namespace"`

	// ConstLabels are values that are applied for every exported metric.
	ConstLabels prometheus.Labels `mapstructure:"const_labels"`

	// SendTimestamps will send the underlying scrape timestamp with the export
	SendTimestamps bool `mapstructure:"send_timestamps"`

	// MetricExpiration defines how long metrics are kept without updates
	MetricExpiration time.Duration `mapstructure:"metric_expiration"`

	// ResourceAttributesAsTags, if sMet to true, will use the exporterhelper feature to
	// transform all resource attributes into metric labels.
	ResourceAttributesAsTags bool `mapstructure:"resource_attributes_as_tags"`
}
