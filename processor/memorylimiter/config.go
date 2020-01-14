// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package memorylimiter provides a processor for OpenTelemetry Service pipeline
// that drops data on the pipeline according to the current state of memory
// usage.
package memorylimiter

import (
	"time"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// Config defines configuration for memory memoryLimiter processor.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// CheckInterval is the time between measurements of memory usage for the
	// purposes of avoiding going over the limits. Defaults to zero, so no
	// checks will be performed.
	CheckInterval time.Duration `mapstructure:"check_interval"`

	// MemoryLimitMiB is the maximum amount of memory, in MiB, targeted to be
	// allocated by the process.
	MemoryLimitMiB uint32 `mapstructure:"limit_mib"`

	// MemorySpikeLimitMiB is the maximum, in MiB, spike expected between the
	// measurements of memory usage.
	MemorySpikeLimitMiB uint32 `mapstructure:"spike_limit_mib"`

	// BallastSizeMiB is the size, in MiB, of the ballast size being used by the
	// process.
	BallastSizeMiB uint32 `mapstructure:"ballast_size_mib"`
}

// Name of BallastSizeMiB config option.
const ballastSizeMibKey = "ballast_size_mib"
