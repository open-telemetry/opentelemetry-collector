// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package batchprocessor

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (c *Config) Validate() error {
	if c.SendBatchMaxSize > 0 && c.SendBatchMaxSize < c.SendBatchSize {
		return errors.New("send_batch_max_size must be greater or equal to send_batch_size")
	}

	uniq := map[string]bool{}
	for _, k := range c.MetadataKeys {
		l := strings.ToLower(k)
		if _, has := uniq[l]; has {
			return fmt.Errorf("duplicate entry in metadata_keys: %q (case-insensitive)", l)
		}
		uniq[l] = true
	}
	if c.Timeout < 0 {
		return errors.New("timeout must be greater or equal to 0")
	}

	// Also do the regular json schema checks
	c.validateSchema()
	return nil
}

// TODO: Autogenerate such a function for every config/
func (c *Config) validateSchema() error {
	buf, err := json.Marshal(c)
	if err != nil {
		return err
	}

	var newConfig Config
	err = newConfig.UnmarshalJSON(buf)
	if err != nil {
		return err
	}
	return nil
}
