// Copyright 2019, OpenTelemetry Authors
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

// Package telemetry controls the telemetry settings to be used in the collector.
package telemetry

import (
	"fmt"
	"strings"
)

const (
	// None indicates that no telemetry data should be collected.
	None Level = iota - 1
	// Basic is the default and covers the basics of the service telemetry.
	Basic
	// Normal adds some other indicators on top of basic.
	Normal
	// Detailed adds dimensions and views to the previous levels.
	Detailed
)

// Level of telemetry data to be generated.
type Level int8

// ParseLevel returns the Level represented by the string. The parsing is case-insensitive
// and it returns error if the string value is unknown.
func ParseLevel(s string) (Level, error) {
	var level Level
	str := strings.ToLower(s)
	switch str {
	case "none":
		level = None
	case "", "basic":
		level = Basic
	case "normal":
		level = Normal
	case "detailed":
		level = Detailed
	default:
		return level, fmt.Errorf("unknown metrics level %q", str)
	}

	return level, nil
}
