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

package expandconverter // import "go.opentelemetry.io/collector/confmap/converter/expandconverter"

import (
	"context"
	"fmt"
	"os"
	"sort"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/internal/nonfatalerror"
	"go.opentelemetry.io/collector/service/featuregate"
)

var (
	// raiseErrorOnUnknownEnvVarFeatureGateID controls whether to raise an error when an environment variable is used but not present.
	raiseErrorOnUnknownEnvVarFeatureGateID = "confmap.expandconverter.RaiseErrorOnUnknownEnvVar"
	raiseErrorOnUnknownEnvVarFeatureGate   = featuregate.Gate{
		ID:          raiseErrorOnUnknownEnvVarFeatureGateID,
		Description: "Raise an error when an environment variable is used but not present",
		Enabled:     false,
	}
)

func init() {
	featuregate.GetRegistry().MustRegister(raiseErrorOnUnknownEnvVarFeatureGate)
}

type converter struct {
	registry *featuregate.Registry
}

type stringSet map[string]struct{}

func (s stringSet) Add(str string) { s[str] = struct{}{} }
func (s stringSet) Merge(other stringSet) {
	for str := range other {
		s[str] = struct{}{}
	}
}
func (s stringSet) Slice() []string {
	var slice []string
	for elem := range s {
		slice = append(slice, elem)
	}
	// sort for reproducibility in unit tests.
	// remove if ever in the hot path.
	sort.Strings(slice)
	return slice
}

// New returns a confmap.Converter, that expands all environment variables for a given confmap.Conf.
//
// Notice: This API is experimental.
func New() confmap.Converter {
	return newWithRegistry(featuregate.GetRegistry())
}

// newWithRegistry is useful for unit tests.
func newWithRegistry(registry *featuregate.Registry) confmap.Converter {
	return &converter{
		registry: registry,
	}
}

func (c converter) Convert(_ context.Context, conf *confmap.Conf) error {
	out := make(map[string]interface{})
	unknownVars := make(stringSet)
	for _, k := range conf.AllKeys() {
		expanded, unknownExpanded := expandStringValues(conf.Get(k))
		out[k] = expanded
		unknownVars.Merge(unknownExpanded)
	}

	unknownVarsSlice := unknownVars.Slice()
	var unknownErr error
	if len(unknownVarsSlice) > 0 {
		unknownErr = fmt.Errorf("failed to expand unknown environment variable(s): %v", unknownVarsSlice)
		if !c.registry.IsEnabled(raiseErrorOnUnknownEnvVarFeatureGateID) {
			unknownErr = nonfatalerror.New(fmt.Errorf(
				"%w. Use %q to turn this into a fatal error",
				unknownErr, raiseErrorOnUnknownEnvVarFeatureGateID,
			))
		}
	}

	if err := conf.Merge(confmap.NewFromStringMap(out)); err != nil {
		return err
	}
	return unknownErr
}

func expandStringValues(value interface{}) (expanded interface{}, unknownVars stringSet) {
	unknownVars = make(stringSet)

	switch v := value.(type) {
	case string:
		return expandEnv(v)
	case []interface{}:
		nslice := make([]interface{}, 0, len(v))
		for _, vint := range v {
			expandedVal, unknownExpanded := expandStringValues(vint)
			nslice = append(nslice, expandedVal)
			unknownVars.Merge(unknownExpanded)
		}
		return nslice, unknownVars
	case map[string]interface{}:
		nmap := map[string]interface{}{}
		for mk, mv := range v {
			expandedVal, unknownExpanded := expandStringValues(mv)
			nmap[mk] = expandedVal
			unknownVars.Merge(unknownExpanded)
		}
		return nmap, unknownVars
	default:
		return v, unknownVars
	}
}

func expandEnv(s string) (expanded string, unknownVars stringSet) {
	unknownVars = make(stringSet)
	expanded = os.Expand(s, func(str string) string {
		// This allows escaping environment variable substitution via $$, e.g.
		// - $FOO will be substituted with env var FOO
		// - $$FOO will be replaced with $FOO
		// - $$$FOO will be replaced with $ + substituted env var FOO
		if str == "$" {
			return "$"
		}

		// Use LookupEnv to check if environment variable exists
		val, ok := os.LookupEnv(str)
		if !ok {
			unknownVars.Add(str)
		}
		return val
	})

	return expanded, unknownVars
}
