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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/mapprovider/envmapprovider"
	"go.opentelemetry.io/collector/config/mapprovider/filemapprovider"
	"go.opentelemetry.io/collector/config/mapprovider/yamlmapprovider"
)

// Deprecated: [v0.48.0] use envmapprovider.New
var NewEnv = envmapprovider.New

// Deprecated: [v0.48.0] use filemapprovider.New
var NewFile = filemapprovider.New

// Deprecated: [v0.48.0] use yamlmapprovider.New
var NewYAML = yamlmapprovider.New

// Deprecated: [v0.48.0] use config.MapProvider
type Provider = config.MapProvider

// Deprecated: [v0.48.0] use config.WatcherFunc
type WatcherFunc = config.WatcherFunc

// Deprecated: [v0.48.0] use config.ChangeEvent
type ChangeEvent = config.ChangeEvent

// Deprecated: [v0.48.0] use config.Retrieved
type Retrieved = config.Retrieved

// Deprecated: [v0.48.0] use config.CloseFunc
type CloseFunc = config.CloseFunc
