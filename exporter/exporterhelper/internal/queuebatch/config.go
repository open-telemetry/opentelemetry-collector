// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
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
	Sizer exporterbatcher.SizerType `mapstructure:"sizer"`

	// QueueSize represents the maximum data size allowed for concurrent storage and processing.
	QueueSize int `mapstructure:"queue_size"`

	// BlockOnOverflow determines the behavior when the component's TotalSize limit is reached.
	// If true, the component will wait for space; otherwise, operations will immediately return a retryable error.
	BlockOnOverflow bool `mapstructure:"block_on_overflow"`

	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue.
	// TODO: This will be changed to Optional when available.
	StorageID *component.ID `mapstructure:"storage"`

	// NumConsumers is the maximum number of concurrent consumers from the queue.
	// This applies across all different optional configurations from above (e.g. wait_for_result, blocking, persistent, etc.).
	// TODO: This will also control the maximum number of shards, when supported:
	//  https://github.com/open-telemetry/opentelemetry-collector/issues/12473.
	NumConsumers int `mapstructure:"num_consumers"`

	// BatchConfig it configures how the requests are consumed from the queue and batch together during consumption.
	// TODO: This will be changed to Optional when available.
	BatchConfig *BatchConfig `mapstructure:"batch"`
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

	if cfg.StorageID != nil && cfg.WaitForResult {
		return errors.New("`wait_for_result` is not supported with a persistent queue configured with `storage`")
	}
	return nil
}

// BatchConfig defines a configuration for batching requests based on a timeout and a minimum number of items.
type BatchConfig struct {
	// FlushTimeout sets the time after which a batch will be sent regardless of its size.
	FlushTimeout time.Duration `mapstructure:"flush_timeout"`

	// MinSize defines the configuration for the minimum size of a batch.
	MinSize int `mapstructure:"min_size"`

	// MaxSize defines the configuration for the maximum size of a batch.
	MaxSize int `mapstructure:"max_size"`
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
