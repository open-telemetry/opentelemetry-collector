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

package zipkinexporter

import (
	"go.opentelemetry.io/collector/config/configmodels"
)

// Config defines configuration settings for the Zipkin exporter.
type Config struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// The URL to send the Zipkin trace data to (e.g.:
	// http://some.url:9411/api/v2/spans).
	URL    string `mapstructure:"url"`
	Format string `mapstructure:"format"`

	// Whether resource labels from TraceData are to be included in Span. True by default
	// This is a temporary flag and will be removed soon,
	// see https://go.opentelemetry.io/collector/issues/595
	ExportResourceLabels *bool `mapstructure:"export_resource_labels"`

	DefaultServiceName string `mapstructure:"default_service_name"`
}
