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

package parserprovider // import "go.opentelemetry.io/collector/service/parserprovider"

// NewDefaultMapProvider returns the default MapProvider, and it creates configuration from a file
// defined by the given configFile and overwrites fields using properties.
func NewDefaultMapProvider(configFileName string, properties []string) MapProvider {
	return NewExpandMapProvider(
		NewMergeMapProvider(
			NewFileMapProvider(configFileName),
			NewPropertiesMapProvider(properties)))
}
