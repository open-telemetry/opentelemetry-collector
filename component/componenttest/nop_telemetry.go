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

package componenttest

import (
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

// NewNopTelemetryCreateSettings returns a new nop telemetry settings for Create* functions.
func NewNopTelemetryCreateSettings() component.TelemetryCreateSettings {
	return component.TelemetryCreateSettings{
		Logger:         zap.NewNop(),
		TracerProvider: trace.NewNoopTracerProvider(),
	}
}
