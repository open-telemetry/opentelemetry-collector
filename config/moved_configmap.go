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

package config // import "go.opentelemetry.io/collector/config"

import (
	"go.opentelemetry.io/collector/confmap"
)

const (
	// Deprecated: [v0.53.0] use confmap.KeyDelimiter
	KeyDelimiter = confmap.KeyDelimiter
)

// Deprecated: [v0.53.0] use confmap.Converter
type MapConverter = confmap.Converter

// Deprecated: [v0.53.0] use confmap.Conf
type Map = confmap.Conf

// Deprecated: [v0.53.0] use confmap.New
var NewMap = confmap.New

// Deprecated: [v0.53.0] use confmap.NewFromStringMap
var NewMapFromStringMap = confmap.NewFromStringMap

// Deprecated: [v0.53.0] use confmap.Provider
type MapProvider = confmap.Provider

// Deprecated: [v0.53.0] use confmap.Retrieved
type Retrieved = confmap.Retrieved

// Deprecated: [v0.53.0] use confmap.RetrievedOption
type RetrievedOption = confmap.RetrievedOption

// Deprecated: [v0.53.0] use confmap.WithRetrievedClose
var WithRetrievedClose = confmap.WithRetrievedClose

// Deprecated: [v0.53.0] use confmap.NewRetrieved
func NewRetrievedFromMap(conf *confmap.Conf, opts ...confmap.RetrievedOption) confmap.Retrieved {
	ret, _ := confmap.NewRetrieved(conf.ToStringMap(), opts...)
	return ret
}

// Deprecated: [v0.53.0] use confmap.WatcherFunc
type WatcherFunc = confmap.WatcherFunc

// Deprecated: [v0.53.0] use confmap.CloseFunc
type CloseFunc = confmap.CloseFunc

// Deprecated: [v0.53.0] use confmap.ChangeEvent
type ChangeEvent = confmap.ChangeEvent
