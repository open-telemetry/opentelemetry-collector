package sqlreceiver // import "go.opentelemetry.io/collector/receiver/sqlreceiver"

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

// MetricConfig defines how to map a column to a metric
type MetricConfig struct {
	Name       string            `mapstructure:"name"`
	ValueField string            `mapstructure:"value_field"`
	MetricType string            `mapstructure:"metric_type"`
	Attributes map[string]string `mapstructure:"attributes"`
}

// LogConfig defines how to map columns to a log record
type LogConfig struct {
	BodyField  string            `mapstructure:"body_field"`
	Attributes map[string]string `mapstructure:"attributes"`
}

// Query defines a single SQL query configuration
type Query struct {
	Name       string          `mapstructure:"name"`
	SQL        string          `mapstructure:"sql"`
	SignalType string          `mapstructure:"signal_type"`
	Metrics    []*MetricConfig `mapstructure:"metrics"` // Used when signal_type is "metrics"
	Logs       *LogConfig      `mapstructure:"logs"`    // Used when signal_type is "logs"
}

// Config defines the configuration for the sqlreceiver
type Config struct {
	confighttp.ClientConfig `mapstructure:",squash"`
	CollectionInterval      time.Duration `mapstructure:"collection_interval"`
	DeploymentInstance      string        `mapstructure:"deployment_instance"`
	HostIP                  string        `mapstructure:"host_ip"`
	// [NEW] Auth Configuration
	AuthEndpoint string `mapstructure:"auth_endpoint"`
	AuthPassword string `mapstructure:"auth_password"`

	Queries []*Query `mapstructure:"queries"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	return c.ClientConfig.Validate()
}
