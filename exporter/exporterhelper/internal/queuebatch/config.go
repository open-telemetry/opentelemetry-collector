// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// Config defines configuration for queueing and batching incoming requests.
type Config struct {
	// Enabled indicates whether to not enqueue and batch before exporting.
	Enabled bool `mapstructure:"enabled"`

	// WaitForResult determines if incoming requests are blocked until the request is processed or not.
	// Currently, this option is not available when persistent queue is configured using the storage configuration.
	WaitForResult bool `mapstructure:"wait_for_result"`

	// Sizer determines the type of size measurement used by this component.
	// It accepts "requests", "items", or "bytes".
	Sizer request.SizerType `mapstructure:"sizer"`

	// QueueSize represents the maximum data size allowed for concurrent storage and processing.
	QueueSize int64 `mapstructure:"queue_size"`

	// BlockOnOverflow determines the behavior when the component's TotalSize limit is reached.
	// If true, the component will wait for space; otherwise, operations will immediately return a retryable error.
	BlockOnOverflow bool `mapstructure:"block_on_overflow"`

	// Deprecated: [v0.123.0] use `block_on_overflow`.
	Blocking bool `mapstructure:"blocking"`

	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue.
	// TODO: This will be changed to Optional when available.
	StorageID *component.ID `mapstructure:"storage"`

	// NumConsumers is the maximum number of concurrent consumers from the queue.
	// This applies across all different optional configurations from above (e.g. wait_for_result, blockOnOverflow, persistent, etc.).
	// TODO: This will also control the maximum number of shards, when supported:
	//  https://github.com/open-telemetry/opentelemetry-collector/issues/12473.
	NumConsumers int `mapstructure:"num_consumers"`

	// BatchConfig it configures how the requests are consumed from the queue and batch together during consumption.
	// TODO: This will be changed to Optional when available.
	Batch *BatchConfig `mapstructure:"batch"`

	// TODO: Remove when deprecated "blocking" is removed.
	hasBlocking bool
}

func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}

	// If user still uses the old blocking, override and will log error during initialization.
	if conf.IsSet("blocking") {
		cfg.hasBlocking = true
		cfg.BlockOnOverflow = cfg.Blocking
	}

	return nil
}

// Validate checks if the Config is valid
func (cfg *Config) Validate() error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.NumConsumers <= 0 {
		return errors.New("`num_consumers` must be positive")
	}

	if cfg.QueueSize <= 0 {
		return errors.New("`queue_size` must be positive")
	}

	// Only support request sizer for persistent queue at this moment.
	if cfg.StorageID != nil && cfg.WaitForResult {
		return errors.New("`wait_for_result` is not supported with a persistent queue configured with `storage`")
	}

	// Only support request sizer for persistent queue at this moment.
	if cfg.StorageID != nil && cfg.Sizer != request.SizerTypeRequests {
		return errors.New("persistent queue configured with `storage` only supports `requests` sizer")
	}

	if cfg.Batch != nil {
		// Only support items or bytes sizer for batch at this moment.
		if cfg.Sizer != request.SizerTypeItems && cfg.Sizer != request.SizerTypeBytes {
			return errors.New("`batch` supports only `items` or `bytes` sizer")
		}

		// Avoid situations where the queue is not able to hold any data.
		if cfg.Batch.MinSize > cfg.QueueSize {
			return errors.New("`min_size` must be less than or equal to `queue_size`")
		}
	}

	return nil
}

// BatchConfig defines a configuration for batching requests based on a timeout and a minimum number of items.
type BatchConfig struct {
	// FlushTimeout sets the time after which a batch will be sent regardless of its size.
	FlushTimeout time.Duration `mapstructure:"flush_timeout"`

	// MinSize defines the configuration for the minimum size of a batch.
	MinSize int64 `mapstructure:"min_size"`

	// MaxSize defines the configuration for the maximum size of a batch.
	MaxSize int64 `mapstructure:"max_size"`
}

func (cfg *BatchConfig) Validate() error {
	if cfg == nil {
		return nil
	}

	if cfg.FlushTimeout <= 0 {
		return errors.New("`flush_timeout` must be positive")
	}

	if cfg.MinSize < 0 {
		return errors.New("`min_size` must be non-negative")
	}

	if cfg.MaxSize < 0 {
		return errors.New("`max_size` must be non-negative")
	}

	if cfg.MaxSize > 0 && cfg.MaxSize < cfg.MinSize {
		return errors.New("`max_size` must be greater or equal to `min_size`")
	}

	return nil
}
