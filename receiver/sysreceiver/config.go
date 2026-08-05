package sysreceiver

import (
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config defines the configuration for the system metrics receiver.
type Config struct {
	// CollectionInterval is the interval at which metrics are collected.
	CollectionInterval time.Duration `mapstructure:"collection_interval"`

	// DeploymentInstance is an identifier for the deployment (optional).
	DeploymentInstance string `mapstructure:"deployment_instance"`

	// NodeValue is the value for the node identifier (e.g. "sundb1").
	NodeValue string `mapstructure:"node_value"`

	// NodeColumnName is the column name for the node identifier (default: "NODE").
	NodeColumnName string `mapstructure:"node_column_name"`

	// HostIP is the server IP (required by user).
	HostIP string `mapstructure:"host_ip"`
}

func createDefaultConfig() component.Config {
	return &Config{
		CollectionInterval: 30 * time.Second,
		NodeValue:          "default-node",
		NodeColumnName:     "NODE",
		HostIP:             "127.0.0.1",
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	return nil
}
