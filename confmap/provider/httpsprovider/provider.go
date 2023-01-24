// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpsprovider // import "go.opentelemetry.io/collector/confmap/provider/httpsprovider"

import (
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal/configurablehttpprovider"
)

// New returns a new confmap.Provider that reads the configuration from a https server.
//
// This Provider supports "https" scheme. One example of an HTTPS URI is: https://localhost:3333/getConfig
//
// To add extra CA certificates you need to install certificates in the system pool. This procedure is operating system
// dependent. E.g.: on Linux please refer to the `update-ca-trust` command.
func New() confmap.Provider {
	return configurablehttpprovider.New(configurablehttpprovider.HTTPSScheme)
}
