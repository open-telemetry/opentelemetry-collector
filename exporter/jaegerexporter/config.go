package jaegerexporter

import "github.com/open-telemetry/opentelemetry-service/configv2/configmodels"

// ConfigV2 defines configuration for Jaeger exporter.
type ConfigV2 struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	CollectorEndpoint             string                   `mapstructure:"collector_endpoint,omitempty"`
	Username                      string                   `mapstructure:"username,omitempty"`
	Password                      string                   `mapstructure:"password,omitempty"`
	ServiceName                   string                   `mapstructure:"service_name,omitempty"`
}
