// Copyright The OpenTelemetry Authors
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

package config // import "go.opentelemetry.io/collector/cmd/builder/internal/config"

import (
	"embed"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/fs"
)

//go:embed *.yaml
var configs embed.FS

// DefaultProvider returns a koanf.Provider that provides the default build
// configuration file. This is the same configuration that otelcorecol is
// built with.
func DefaultProvider() koanf.Provider {
	return fs.Provider(configs, "default.yaml")
}
