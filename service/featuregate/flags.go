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

package featuregate // import "go.opentelemetry.io/collector/service/featuregate"

import (
	"flag"
	"sort"
	"strings"
)

const gatesListCfg = "feature-gates"

var gatesList = FlagValue{}

// Flags adds CLI flags for managing feature gates to the provided FlagSet
// Feature gates can be configured with `--feature-gates=foo,-bar`.  This would
// enable the `foo` feature gate and disable the `bar` feature gate.
func Flags(flags *flag.FlagSet) {
	flags.Var(
		gatesList,
		gatesListCfg,
		"Comma-delimited list of feature gate identifiers. Prefix with '-' to disable the feature.  '+' or no prefix will enable the feature.")
}

// GetFlags returns the FlagValue used with Flags()
func GetFlags() FlagValue {
	return gatesList
}

var _ flag.Value = (*FlagValue)(nil)

// FlagValue implements the flag.Value interface and provides a mechanism for applying feature
// gate statuses to a Registry
type FlagValue map[string]bool

// String returns a string representing the FlagValue
func (f FlagValue) String() string {
	var t []string
	for k, v := range f {
		if v {
			t = append(t, k)
		} else {
			t = append(t, "-"+k)
		}
	}

	// Sort the list of identifiers for consistent results
	sort.Strings(t)
	return strings.Join(t, ",")
}

// Set applies the FlagValue encoded in the input string
func (f FlagValue) Set(s string) error {
	return f.SetSlice(strings.Split(s, ","))
}

// SetSlice applies the feature gate statuses in the input slice to the FlagValue
func (f FlagValue) SetSlice(s []string) error {
	for _, v := range s {
		var id string
		var val bool
		switch v[0] {
		case '-':
			id = v[1:]
			val = false
		case '+':
			id = v[1:]
			val = true
		default:
			id = v
			val = true
		}

		if _, exists := f[id]; exists {
			// If the status has already been set, ignore it
			// This allows CLI flags, which are processed first
			// to take precedence over config settings
			continue
		}
		f[id] = val
	}

	return nil
}
