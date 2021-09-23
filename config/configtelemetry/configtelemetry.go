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
	"flag"
	"fmt"
	"strings"
)

const (
	// LevelNone indicates that no telemetry data should be collected.
	LevelNone Level = iota - 1
	// LevelBasic is the recommended and covers the basics of the service telemetry.
	LevelBasic
	// LevelNormal adds some other indicators on top of basic.
	LevelNormal
	// LevelDetailed adds dimensions and views to the previous levels.
	LevelDetailed

	levelNoneStr     = "none"
	levelBasicStr    = "basic"
	levelNormalStr   = "normal"
	levelDetailedStr = "detailed"

	metricsLevelCfg = "metrics-level"
)

var metricsLevelPtr = new(Level)

// Flags is a helper function to add telemetry config flags to the service that exposes
// the application flags.
func Flags(flags *flag.FlagSet) {
	flags.Var(
		metricsLevelPtr,
		metricsLevelCfg,
		"Output level of telemetry metrics (none, basic, normal, detailed)")
}

// Level is the level of internal telemetry (metrics, logs, traces about the component itself)
// that every component should generate.
type Level int8

var _ flag.Value = (*Level)(nil)

func (l *Level) String() string {
	switch *l {
	case LevelNone:
		return levelNoneStr
	case LevelBasic:
		return levelBasicStr
	case LevelNormal:
		return levelNormalStr
	case LevelDetailed:
		return levelDetailedStr
	}
	return "unknown"
}

// Set sets the telemetry level.
func (l *Level) Set(s string) error {
	lvl, err := parseLevel(s)
	if err != nil {
		return err
	}
	*l = lvl
	return nil
}

// GetMetricsLevelFlagValue returns the value of the "--metrics-level" flag.
// IMPORTANT: This must be used only in the core collector code for the moment.
func GetMetricsLevelFlagValue() Level {
	return *metricsLevelPtr
}

// TelemetrySetting exposes the common Telemetry configuration for one component.
type TelemetrySetting struct {
	// MetricsLevelStr is the level of telemetry metrics, the possible values are:
	//  - "none" indicates that no telemetry data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	//  - "normal" adds some other indicators on top of basic.
	//  - "detailed" adds dimensions and views to the previous levels.
	MetricsLevelStr string `mapstructure:"metrics_level"`
}

// DefaultTelemetrySetting returns the default TelemetrySetting.
// The level is set to the "--metrics-level" flag if set, otherwise the default "basic" level.
func DefaultTelemetrySetting() TelemetrySetting {
	return TelemetrySetting{
		MetricsLevelStr: metricsLevelPtr.String(),
	}
}

// GetMetricsLevel returns the parsed level, or error if unknown value.
// Empty string is consider unknown value.
func (ts TelemetrySetting) GetMetricsLevel() (Level, error) {
	return parseLevel(ts.MetricsLevelStr)
}

// ParseLevel returns the Level represented by the string. The parsing is case-insensitive
// and it returns error if the string value is unknown.
func parseLevel(str string) (Level, error) {
	str = strings.ToLower(str)

	switch str {
	case levelNoneStr:
		return LevelNone, nil
	case levelBasicStr:
		return LevelBasic, nil
	case levelNormalStr:
		return LevelNormal, nil
	case levelDetailedStr:
		return LevelDetailed, nil
	}
	return LevelNone, fmt.Errorf("unknown metrics level %q", str)
}
