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

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporters/otlpgrpcexporter"
)

// Deprecated: [v0.63.0], use `otlpgrpcexporter.Config` instead.
// Config defines configuration for OTLP exporter.
type Config = otlpgrpcexporter.Config

// Deprecated: [v0.63.0], use `otlpgrpcexporter.NewFactory` instead.
// NewFactory creates a factory for OTLP exporter.
func NewFactory() component.ExporterFactory {
	return otlpgrpcexporter.NewFactory()
}
