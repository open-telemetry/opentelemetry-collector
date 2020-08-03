package cortexexporter

import (
	prometheus_golang "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for Remote Write exporter.
type Config struct {
	configmodels.ExporterSettings  `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	exporterhelper.RetrySettings   `mapstructure:"retry_on_failure"`

	// Namespace if set, exports metrics under the provided value.
	Namespace string `mapstructure:"namespace"`

	// ConstLabels are values that are applied for every exported metric.
	ConstLabels prometheus_golang.Labels `mapstructure:"const_labels"`

	HTTPClientSettings confighttp.HTTPClientSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

}
