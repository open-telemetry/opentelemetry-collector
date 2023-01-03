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

// Package configopaque implements String type alias to mask sensitive information.
// Use configopaque.String on the type of sensitive fields, to mask the
// opaque string as `[REDACTED]`.
//
// This ensure that no sensitive information is leaked when printing the
// full Collector configurations.
package configopaque // import "go.opentelemetry.io/collector/config/configopaque"
