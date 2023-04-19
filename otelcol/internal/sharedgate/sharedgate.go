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

// Package sharedgate exposes a featuregate that is used by multiple packages.
package sharedgate // import "go.opentelemetry.io/collector/otelcol/internal/sharedgate"

import "go.opentelemetry.io/collector/featuregate"

var ConnectorsFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"service.connectors",
	featuregate.StageStable,
	featuregate.WithRegisterFromVersion("v0.71.0"),
	featuregate.WithRegisterDescription("Enables 'connectors', a new type of component for transmitting signals between pipelines."),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/issues/2336"),
	featuregate.WithRegisterToVersion("v0.78.0"))
