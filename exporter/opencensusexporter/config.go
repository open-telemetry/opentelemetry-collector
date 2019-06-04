// Copyright 2019, OpenCensus Authors
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

	"github.com/census-instrumentation/opencensus-service/internal/configmodels"
)

// ConfigV2 defines configuration for OpenCensus exporter.
type ConfigV2 struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	Endpoint                      string                   `mapstructure:"endpoint"`
	Compression                   string                   `mapstructure:"compression"`
	Headers                       map[string]string        `mapstructure:"headers"`
	NumWorkers                    int                      `mapstructure:"num-workers"`
	CertPemFile                   string                   `mapstructure:"cert-pem-file"`
	UseSecure                     bool                     `mapstructure:"secure,omitempty"`
	ReconnectionDelay             time.Duration            `mapstructure:"reconnection-delay,omitempty"`
	KeepaliveParameters           *keepaliveConfig         `mapstructure:"keepalive,omitempty"`
}
