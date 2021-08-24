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

package kafkaexporter

import (
	"time"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for Kafka exporter.
type Config struct {
	config.ExporterSettings        `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	exporterhelper.RetrySettings   `mapstructure:"retry_on_failure"`

	// The list of kafka brokers (default localhost:9092)
	Brokers []string `mapstructure:"brokers"`
	// Kafka protocol version
	ProtocolVersion string `mapstructure:"protocol_version"`
	// The name of the kafka topic to export to (default otlp_spans for traces, otlp_metrics for metrics)
	Topic string `mapstructure:"topic"`

	// Encoding of messages (default "otlp_proto")
	Encoding string `mapstructure:"encoding"`

	// Metadata is the namespace for metadata management properties used by the
	// Client, and shared by the Producer/Consumer.
	Metadata Metadata `mapstructure:"metadata"`

	// Producer is the namespaces for producer properties used only by the Producer
	Producer Producer `mapstructure:"producer"`

	// Authentication defines used authentication mechanism.
	Authentication Authentication `mapstructure:"auth"`
}

// Metadata defines configuration for retrieving metadata from the broker.
type Metadata struct {
	// Whether to maintain a full set of metadata for all topics, or just
	// the minimal set that has been necessary so far. The full set is simpler
	// and usually more convenient, but can take up a substantial amount of
	// memory if you have many topics and partitions. Defaults to true.
	Full bool `mapstructure:"full"`

	// Retry configuration for metadata.
	// This configuration is useful to avoid race conditions when broker
	// is starting at the same time as collector.
	Retry MetadataRetry `mapstructure:"retry"`
}

// Producer defines configuration for producer
type Producer struct {
	// Maximum message bytes the producer will accept to produce.
	MaxMessageBytes int `mapstructure:"max_message_bytes"`
}

// MetadataRetry defines retry configuration for Metadata.
type MetadataRetry struct {
	// The total number of times to retry a metadata request when the
	// cluster is in the middle of a leader election or at startup (default 3).
	Max int `mapstructure:"max"`
	// How long to wait for leader election to occur before retrying
	// (default 250ms). Similar to the JVM's `retry.backoff.ms`.
	Backoff time.Duration `mapstructure:"backoff"`
}

var _ config.Exporter = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	return nil
}
