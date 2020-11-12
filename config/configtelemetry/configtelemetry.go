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

package configtelemetry

import (
	"fmt"
	"strings"
)

const (
	// LevelNone indicates that no telemetry data should be collected.
	LevelNone Level = iota - 1
	// LevelBasic is the default and covers the basics of the service telemetry.
	LevelBasic
	// LevelNormal adds some other indicators on top of basic.
	LevelNormal
	// LevelDetailed adds dimensions and views to the previous levels.
	LevelDetailed
)

// Level is the level of internal telemetry (metrics, logs, traces about the component itself)
// that every component should generate.
type Level int8

// ParseLevel returns the Level represented by the string. The parsing is case-insensitive
// and it returns error if the string value is unknown.
func ParseLevel(str string) (Level, error) {
	str = strings.ToLower(str)

	switch str {
	case "none":
		return LevelNone, nil
	case "basic":
		return LevelBasic, nil
	case "normal":
		return LevelNormal, nil
	case "detailed":
		return LevelDetailed, nil
	}
	return LevelNone, fmt.Errorf("unknown metrics level %q", str)
}
