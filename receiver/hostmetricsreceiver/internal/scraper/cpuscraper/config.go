// Copyright 2020, OpenTelemetry Authors
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

package cpuscraper

import "github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"

// Config relating to CPU Metric Scraper.
type Config struct {
	internal.ConfigSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// If `true`, stats will be generated for the system as a whole _as well
	// as_ for each individual CPU/core in the system and will be distinguished
	// by the `cpu` dimension.  If `false`, stats will only be generated for
	// the system as a whole that will not include a `cpu` dimension.
	ReportPerCPU bool `mapstructure:"report_per_cpu"`
}
