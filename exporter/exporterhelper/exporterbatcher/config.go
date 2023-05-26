// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"errors"
	"time"
)

// Config defines configuration for queueing batches before sending to the consumerSender.
type Config struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`

	// MinSizeMib is the size when the batch should be sent regardless of the timeout.
	// There is no guarantee that the batch size always greater than this value.
	MinSizeMib int `mapstructure:"min_size_mib"`
	// MaxSizeMib is the maximum size of a batch in MiB. If the batch size exceeds this value, it will be broken up
	// into smaller batches if possible. If set, it's guaranteed that the batch size is less than or equal to this value.
	MaxSizeMib int `mapstructure:"max_size_mib"`

	// MinSizeItems is the number of items (spans, data points or log records) when the batch should be sent regardless
	// of the timeout. There is no guarantee that the batch size always greater than this value.
	MinSizeItems int `mapstructure:"min_size_items"`
	// MaxSizeItems is the maximum number of the batch items, i.e. spans, data points or log records for OTLP,
	// but can be anything else for other formats. If the batch size exceeds this value,
	// it will be broken up into smaller batches if possible.
	MaxSizeItems int `mapstructure:"max_size_items"`

	// Timeout sets the time after which a batch will be sent regardless of its size.
	// When this is set to zero, batched data will be sent immediately.
	// This is a recommended option, as it will ensure that the data is sent in a timely manner.
	Timeout time.Duration `mapstructure:"timeout"` // Is there a better name to avoid confusion with the consumerSender timeout?

	// BatchersLimit is the maximum number of batchers that can be created.
	// If this limit is reached, the consumer will start dropping data for the new batch identifiers.
	BatchersLimit int `mapstructure:"batchers_limit"`
}

func (bc *Config) Validate() error {
	if !bc.Enabled {
		return nil
	}
	if bc.MinSizeMib < 0 {
		return errors.New("min_size_mib cannot be negative")
	}
	if bc.MaxSizeMib < 0 {
		return errors.New("max_size_mib cannot be negative")
	}
	if bc.MinSizeItems < 0 {
		return errors.New("min_size_items cannot be negative")
	}
	if bc.MaxSizeItems < 0 {
		return errors.New("max_size_items cannot be negative")
	}
	if bc.MinSizeMib > bc.MaxSizeMib {
		return errors.New("min_size_mib cannot be greater than max_size_mib")
	}
	if bc.MinSizeItems > bc.MaxSizeItems {
		return errors.New("min_size_items cannot be greater than max_size_items")
	}
	if bc.MinSizeMib == 0 && bc.MaxSizeMib == 0 && bc.MinSizeItems == 0 && bc.MaxSizeItems == 0 {
		return errors.New("batching is enabled but no limits are set")
	}
	return nil
}
