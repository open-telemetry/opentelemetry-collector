// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}

	// If all of the following hold:
	// 1. the sizer is set,
	// 2. the batch sizer is not set,
	// 3. the batch section is nonempty, and
	// 4. the batch Optional has a value,
	// then use the same value as the queue sizer.
	if conf.IsSet("sizer") && !conf.IsSet("batch::sizer") && conf.IsSet("batch") && conf.Get("batch") != nil && cfg.Batch.HasValue() {
		cfg.Batch.Get().Sizer = cfg.Sizer
	}
	return nil
}

// Validate checks if the Config is valid
func validateConfig(cfg *Config) error {
	// Only support request sizer for persistent queue at this moment.
	if cfg.StorageID != nil && cfg.WaitForResult {
		return errors.New("`wait_for_result` is not supported with a persistent queue configured with `storage`")
	}

	if cfg.Batch.HasValue() && cfg.Batch.Get().Sizer == cfg.Sizer {
		// Avoid situations where the queue is not able to hold any data.
		if cfg.Batch.Get().MinSize > cfg.QueueSize {
			return errors.New("`min_size` must be less than or equal to `queue_size`")
		}
	}

	return nil
}

func getDefaultSizer() request.SizerType {
	return request.SizerTypeItems
}

func validateBatchConfig(cfg *BatchConfig) error {
	// Only support items or bytes sizer for batch at this moment.
	if cfg.Sizer != request.SizerTypeItems && cfg.Sizer != request.SizerTypeBytes {
		return fmt.Errorf("`batch` supports only `items` or `bytes` sizer, found %q", cfg.Sizer.String())
	}

	if cfg.MaxSize > 0 && cfg.MaxSize < cfg.MinSize {
		return fmt.Errorf("`max_size` (%d) must be greater or equal to `min_size` (%d)", cfg.MaxSize, cfg.MinSize)
	}

	return nil
}

// Validate metadata_keys for duplicates (case-insensitive)
func validateMetadataKeys(metadataKeys []string) error {
	uniq := map[string]bool{}
	for _, k := range metadataKeys {
		l := strings.ToLower(k)
		if _, has := uniq[l]; has {
			return fmt.Errorf("duplicate entry in metadata_keys: %q (case-insensitive)", l)
		}
		uniq[l] = true
	}
	return nil
}
