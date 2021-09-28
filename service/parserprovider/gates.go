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

package parserprovider

import (
	"context"
	"flag"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/config"
)

var _ flag.Value = (*FlagValue)(nil)

// FlagValue implements the flag.Value interface and provides a mechanism for applying feature
// gate statuses to a Registry.
type FlagValue map[string]bool

// String returns a string representing the FlagValue.
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

// Set applies the FlagValue encoded in the input string.
func (f FlagValue) Set(s string) error {
	for id, state := range unmarshalGates(strings.Split(s, ",")) {
		f[id] = state
	}
	return nil
}

type gatesFlagMapProvider struct{}

// NewGatesMapProvider returns a MapProvider, that provides a config.Map from feature gate properties.
//
// The implementation reads --feature-gates flag(s) from the cmd and sets them in a config.Map
func NewGatesMapProvider() MapProvider {
	return &gatesFlagMapProvider{}
}

func (g *gatesFlagMapProvider) Get(context.Context) (*config.Map, error) {
	gates := getGateFlag()
	if len(gates) == 0 {
		return config.NewMap(), nil
	}

	cfg := map[string]interface{}{}
	for id, state := range gates {
		cfg["service::gates::"+id] = state
	}
	return config.NewMapFromStringMap(cfg), nil
}

func (g *gatesFlagMapProvider) Close(context.Context) error {
	return nil
}

func unmarshalGates(gates []string) map[string]bool {
	ret := make(map[string]bool)
	for _, v := range gates {
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

		if _, exists := ret[id]; exists {
			continue
		}
		ret[id] = val
	}

	return ret
}
