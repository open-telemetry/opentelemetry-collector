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

package kafkametricsreceiver

import (
	"go.opentelemetry.io/collector/exporter/kafkaexporter"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

// Config represents user settings for kafkametrics receiver
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`

	// The list of kafka brokers (default localhost:9092)
	Brokers []string `mapstructure:"brokers"`

	// ProtocolVersion Kafka protocol version
	ProtocolVersion string `mapstructure:"protocol_version"`

	// TopicMatch topics to collect metrics on
	TopicMatch string `mapstructure:"topic_match"`

	// GroupMatch consumer groups to collect on
	GroupMatch string `mapstructure:"group_match"`

	// Authentication data
	Authentication kafkaexporter.Authentication `mapstructure:"auth"`

	// Scrapers defines which metric data points to be captured from kafka
	Scrapers []string `mapstructure:"scrapers"`

	// ClientID is the id associated with the consumer that reads from topics in kafka.
	ClientID string `mapstructure:"client_id"`
}
