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

package jaegerthrifthttpexporter

import (
	"time"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// Config defines configuration for Jaeger Thrift over HTTP exporter.
type Config struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// URL is the URL to send the Jaeger trace data to (e.g.:
	// http://some.url:14268/api/traces).
	URL string `mapstructure:"url"`

	// Timeout is the maximum timeout for HTTP request sending trace data. The
	// default value is 5 seconds.
	Timeout time.Duration `mapstructure:"timeout"`

	// Headers are a set of headers to be added to the HTTP request sending
	// trace data.
	Headers map[string]string `mapstructure:"headers"`
}
