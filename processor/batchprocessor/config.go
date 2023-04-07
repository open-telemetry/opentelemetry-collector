// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config defines configuration for batch processor.
type Config struct {
	// Timeout sets the time after which a batch will be sent regardless of size.
	// When this is set to zero, batched data will be sent immediately.
	Timeout time.Duration `mapstructure:"timeout"`

	// SendBatchSize is the size of a batch which after hit, will trigger it to be sent.
	// SendBatchSize of 0 implies ignoring timeout, data will be sent immediately
	// subject to only send_batch_max_size.
	SendBatchSize uint32 `mapstructure:"send_batch_size"`

	// SendBatchMaxSize is the maximum size of a batch. It must be larger than SendBatchSize.
	// Larger batches are split into smaller units.
	// Default value is 0, that means no maximum size.
	SendBatchMaxSize uint32 `mapstructure:"send_batch_max_size"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.SendBatchMaxSize > 0 && cfg.SendBatchMaxSize < cfg.SendBatchSize {
		return errors.New("send_batch_max_size must be greater or equal to send_batch_size")
	}
	if cfg.Timeout < 0 {
		return errors.New("timeout must be greater or equal to 0")
	}
	return nil
}
