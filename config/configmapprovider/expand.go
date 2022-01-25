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
	"context"
	"os"

	"go.opentelemetry.io/collector/config"
)

// NewExpandConverter returns a service.ConfigMapConverterFunc, that expands all environment variables for a given config.Map.
//
// Notice: This API is experimental.
func NewExpandConverter() config.MapConverterFunc {
	return func(_ context.Context, cfgMap *config.Map) error {
		for _, k := range cfgMap.AllKeys() {
			cfgMap.Set(k, expandStringValues(cfgMap.Get(k)))
		}
		return nil
	}
}

func expandStringValues(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return expandEnv(v)
	case []interface{}:
		nslice := make([]interface{}, 0, len(v))
		for _, vint := range v {
			nslice = append(nslice, expandStringValues(vint))
		}
		return nslice
	case map[string]interface{}:
		nmap := map[string]interface{}{}
		for mk, mv := range v {
			nmap[mk] = expandStringValues(mv)
		}
		return nmap
	default:
		return v
	}
}

func expandEnv(s string) string {
	return os.Expand(s, func(str string) string {
		// This allows escaping environment variable substitution via $$, e.g.
		// - $FOO will be substituted with env var FOO
		// - $$FOO will be replaced with $FOO
		// - $$$FOO will be replaced with $ + substituted env var FOO
		if str == "$" {
			return "$"
		}
		return os.Getenv(str)
	})
}
