package avitoquotaprocessor

import (
	"crypto/tls"
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"
)

type Config struct {
	// One of the values listed in zapcore.Level.
	LogLevel string `mapstructure:"loglevel"`

	RetryRulesOnStartCount int  `mapstructure:"retry_rules_on_start_count"`
	StartWithoutRules      bool `mapstructure:"start_without_rules"`

	RulesUpdateInterval time.Duration `mapstructure:"rules_update_interval"`
	DefaultTTLDays      int           `mapstructure:"default_ttl_days"`
	RateLimiterDryRun   bool          `mapstructure:"rate_limiter_dry_run"`

	ClientTls *ClientTls `mapstructure:"client_tls"`
}

func (c *Config) Validate() error {
	if c.ClientTls == nil {
		return fmt.Errorf("avitoquotaprocessor client tls config is required")
	}
	err := c.ClientTls.Validate()
	if err != nil {
		return err
	}
	return c.validateLogLevel()
}

func (c *Config) validateLogLevel() error {
	var level zapcore.Level
	err := level.Set(c.LogLevel)

	return err
}

type ClientTls struct {
	Enabled bool   `mapstructure:"enabled"`
	CrtPath string `mapstructure:"crt_path"`
	KeyPath string `mapstructure:"key_path"`
}

func (c *ClientTls) Validate() error {
	if !c.Enabled {
		return nil
	}
	_, err := tls.LoadX509KeyPair(c.CrtPath, c.KeyPath)
	if err != nil {
		return fmt.Errorf("invalid tls config: %w", err)
	}
	return nil
}
