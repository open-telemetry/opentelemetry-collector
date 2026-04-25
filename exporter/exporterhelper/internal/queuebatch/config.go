// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// Config defines configuration for queueing and batching incoming requests.
type Config struct {
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

	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue.
	// TODO: This will be changed to Optional when available.
	// See https://github.com/open-telemetry/opentelemetry-collector/issues/13822
	StorageID *component.ID `mapstructure:"storage"`

	// NumConsumers is the maximum number of concurrent consumers from the queue.
	// This applies across all different optional configurations from above (e.g. wait_for_result, block_on_overflow, storage, etc.).
	NumConsumers int `mapstructure:"num_consumers"`

	// BatchConfig it configures how the requests are consumed from the queue and batch together during consumption.
	Batch configoptional.Optional[BatchConfig] `mapstructure:"batch"`
}

func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}

	if conf.IsSet("batch") && conf.IsSet("batch::sizers") {
		if conf.IsSet("batch::min_size") || conf.IsSet("batch::max_size") || conf.IsSet("batch::sizer") {
			return errors.New("cannot specify both `sizers` and legacy fields (`min_size`, `max_size`, `sizer`) in `batch`")
		}
	}

	// If all of the following hold:
	// 1. the sizer is set,
	// 2. the batch sizer is not set and
	// 3. the batch section is nonempty,
	// then use the same value as the queue sizer.
	if conf.IsSet("sizer") && !conf.IsSet("batch::sizer") && !conf.IsSet("batch::sizers") && conf.IsSet("batch") && conf.Get("batch") != nil {
		cfg.Batch.Get().Sizer = cfg.Sizer
	}
	return nil
}

// Validate checks if the Config is valid
func (cfg *Config) Validate() error {
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

	if cfg.Batch.HasValue() {
		batchCfg := cfg.Batch.Get()
		// Check legacy fields
		if batchCfg.Sizer == cfg.Sizer {
			if batchCfg.MinSize > cfg.QueueSize {
				return fmt.Errorf("for sizer %s, `min_size` (%d) must be less than or equal to `queue_size` (%d)", cfg.Sizer.String(), batchCfg.MinSize, cfg.QueueSize)
			}
		}
		// If Sizers map includes the same sizer type as the main sizer, check that the min size is not greater than queue size.
		if limit, ok := batchCfg.Sizers[cfg.Sizer]; ok {
			if limit.MinSize > cfg.QueueSize {
				return fmt.Errorf("for sizer %s, `min_size` (%d) must be less than or equal to `queue_size` (%d)", cfg.Sizer.String(), limit.MinSize, cfg.QueueSize)
			}
		}
	}

	return nil
}

// BatchConfig defines a configuration for batching requests based on a timeout and a minimum number of items.
type BatchConfig struct {
	// FlushTimeout sets the time after which a batch will be sent regardless of its size.
	FlushTimeout time.Duration `mapstructure:"flush_timeout"`

	// Sizer determines the type of size measurement used by the batch.
	// If not configured, use the same configuration as the queue.
	// It accepts "requests", "items", or "bytes".
	//
	// Deprecated: Use Sizers instead.
	Sizer request.SizerType `mapstructure:"sizer"`

	// MinSize defines the configuration for the minimum size of a batch.
	//
	// Deprecated: Use Sizers instead.
	MinSize int64 `mapstructure:"min_size"`

	// MaxSize defines the configuration for the maximum size of a batch.
	//
	// Deprecated: Use Sizers instead.
	MaxSize int64 `mapstructure:"max_size"`

	// Sizers allows configuring multiple sizing constraints.
	// The key is the SizerType (e.g., "bytes", "items").
	Sizers map[request.SizerType]SizerLimit `mapstructure:"sizers"`

	// Partition defines the partitioning of the batches configuration.
	Partition PartitionConfig `mapstructure:"partition"`
}

// SizerLimit defines the configuration for the minimum and maximum size of a batch for a specific sizer.
type SizerLimit struct {
	MinSize int64 `mapstructure:"min_size"`
	MaxSize int64 `mapstructure:"max_size"`
}

// PartitionConfig defines a configuration for partitioning requests based on metadata keys.
type PartitionConfig struct {
	// MetadataKeys is a list of client.Metadata keys that will be used to partition
	// the data into batches. If this setting is empty, a single batcher instance
	// will be used. When this setting is not empty, one batcher will be used per
	// distinct combination of values for the listed metadata keys.
	//
	// Empty value and unset metadata are treated as distinct cases.
	//
	// Entries are case-insensitive. Duplicated entries will trigger a validation error.
	MetadataKeys []string `mapstructure:"metadata_keys"`
}

func (cfg *BatchConfig) Validate() error {
	if cfg == nil {
		return nil
	}

	if cfg.Sizer.String() != "" && len(cfg.Sizers) > 0 {
		return errors.New("both `sizer` and `sizers` are specified, but only one is allowed, `sizers` is preferred")
	}

	if len(cfg.Sizers) == 0 {
		if cfg.Sizer != request.SizerTypeItems && cfg.Sizer != request.SizerTypeBytes {
			return fmt.Errorf("`batch` supports only `items` or `bytes` sizer, found %q", cfg.Sizer.String())
		}
		if cfg.MinSize < 0 {
			return fmt.Errorf("`min_size` must be non-negative, found %d", cfg.MinSize)
		}
		if cfg.MaxSize < 0 {
			return fmt.Errorf("`max_size` must be non-negative, found %d", cfg.MaxSize)
		}
		if cfg.MaxSize > 0 && cfg.MaxSize < cfg.MinSize {
			return fmt.Errorf("`max_size` (%d) must be greater or equal to `min_size` (%d)", cfg.MaxSize, cfg.MinSize)
		}
	}

	if cfg.FlushTimeout <= 0 {
		return fmt.Errorf("`flush_timeout` must be positive, found %d", cfg.FlushTimeout)
	}

	for szt, limit := range cfg.Sizers {
		if szt != request.SizerTypeItems && szt != request.SizerTypeBytes {
			return fmt.Errorf("`batch` supports only `items` or `bytes` sizer, found %q", szt.String())
		}
		if limit.MinSize < 0 {
			return fmt.Errorf("`min_size` must be non-negative for sizer %q, found %d", szt.String(), limit.MinSize)
		}
		if limit.MaxSize < 0 {
			return fmt.Errorf("`max_size` must be non-negative for sizer %q, found %d", szt.String(), limit.MaxSize)
		}
		if limit.MaxSize > 0 && limit.MaxSize < limit.MinSize {
			return fmt.Errorf("`max_size` (%d) must be greater or equal to `min_size` (%d) for sizer %q", limit.MaxSize, limit.MinSize, szt.String())
		}
	}

	return nil
}

func (cfg *PartitionConfig) Validate() error {
	if cfg == nil {
		return nil
	}

	// Validate metadata_keys for duplicates (case-insensitive)
	uniq := map[string]bool{}
	for _, k := range cfg.MetadataKeys {
		l := strings.ToLower(k)
		if _, has := uniq[l]; has {
			return fmt.Errorf("duplicate entry in metadata_keys: %q (case-insensitive)", l)
		}
		uniq[l] = true
	}

	return nil
}
