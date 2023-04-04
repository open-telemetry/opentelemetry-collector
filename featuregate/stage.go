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

package featuregate // import "go.opentelemetry.io/collector/featuregate"

// Stage represents the Gate's lifecycle and what is the expected state of it.
type Stage int8

const (
	// StageAlpha is used when creating a new feature and the Gate must be explicitly enabled
	// by the operator.
	//
	// The Gate will be disabled by default.
	StageAlpha Stage = iota
	// StageBeta is used when the feature gate is well tested and is enabled by default,
	// but can be disabled by a Gate.
	//
	// The Gate will be enabled by default.
	StageBeta
	// StageStable is used when feature is permanently enabled and can not be disabled by a Gate.
	// This value is used to provide feedback to the user that the gate will be removed in the next versions.
	//
	// The Gate will be enabled by default and will return an error if modified.
	StageStable
	// StageDeprecated is used when feature is permanently disabled and can not be enabled by a Gate.
	// This value is used to provide feedback to the user that the gate will be removed in the next versions.
	//
	// The Gate will be disabled by default and will return an error if modified.
	StageDeprecated
)

func (s Stage) String() string {
	switch s {
	case StageAlpha:
		return "Alpha"
	case StageBeta:
		return "Beta"
	case StageStable:
		return "Stable"
	case StageDeprecated:
		return "Deprecated"
	}
	return "Unknown"
}
