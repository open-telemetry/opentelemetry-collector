// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
)

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.SendBatchMaxSize > 0 && cfg.SendBatchMaxSize < cfg.SendBatchSize {
		return errors.New("send_batch_max_size must be greater or equal to send_batch_size")
	}
	uniq := map[string]bool{}
	for _, k := range cfg.MetadataKeys {
		l := strings.ToLower(k)
		if _, has := uniq[l]; has {
			return fmt.Errorf("duplicate entry in metadata_keys: %q (case-insensitive)", l)
		}
		uniq[l] = true
	}
	if cfg.Timeout < 0 {
		return errors.New("timeout must be greater or equal to 0")
	}
	return nil
}
