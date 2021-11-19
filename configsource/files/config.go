package files

import (
	"go.opentelemetry.io/collector/config"
)

// Config defines configuration for logging exporter.
type Config struct {
	config.ConfigSourceSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	Name                        string
}
