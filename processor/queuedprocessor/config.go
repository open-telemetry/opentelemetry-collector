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

package queuedprocessor

import (
	"time"

	"go.opentelemetry.io/collector/config/configmodels"
)

// Config defines configuration for Attributes processor.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// NumConsumers is the number of queue workers that dequeue batches and send them out.
	NumWorkers int `mapstructure:"num_workers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `mapstructure:"queue_size"`
	// Retry indicates whether queue processor should retry span batches in case of processing failure.
	RetryOnFailure bool `mapstructure:"retry_on_failure"`
	// BackoffDelay is the amount of time a worker waits after a failed send before retrying.
	BackoffDelay time.Duration `mapstructure:"backoff_delay"`
}
