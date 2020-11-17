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

package main

import "go.opentelemetry.io/collector/service/defaultcomponents"

// allCfgs creates config yaml metadata files for all registered components
func allCfgs() {
	components, err := defaultcomponents.Components()
	if err != nil {
		panic(err)
	}
	for _, f := range components.Receivers {
		genMeta(f.CreateDefaultConfig())
	}
	for _, f := range components.Extensions {
		genMeta(f.CreateDefaultConfig())
	}
	for _, f := range components.Processors {
		genMeta(f.CreateDefaultConfig())
	}
	for _, f := range components.Exporters {
		genMeta(f.CreateDefaultConfig())
	}
}
