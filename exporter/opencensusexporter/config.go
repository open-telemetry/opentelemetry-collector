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

package opencensusexporter

import (
	"time"

	"github.com/open-telemetry/opentelemetry-collector/config/configgrpc"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// Config defines configuration for OpenCensus exporter.
type Config struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	configgrpc.GRPCSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// The number of workers that send the gRPC requests.
	NumWorkers int `mapstructure:"num_workers"`

	// The time period between each reconnection performed by the exporter.
	ReconnectionDelay time.Duration `mapstructure:"reconnection_delay,omitempty"`
}
