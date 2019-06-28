package summaryapireceiver

import (
	"time"

	"github.com/open-telemetry/opentelemetry-service/internal/configmodels"
)

// ConfigV2 defines configuration for SummaryApi receiver.
type ConfigV2 struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	scrapeInterval                time.Duration            `mapstructure:"scrape_interval"`
	kubeletEndpoint               string                   `mapstructure:"kubelet_endpoint"`
	metricPrefix                  string                   `mapstructure:"metric_prefix"`
}
