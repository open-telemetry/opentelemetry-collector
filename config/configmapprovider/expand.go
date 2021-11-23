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
	"fmt"
	"os"
)

type expandMapProvider struct {
	base Provider
}

// NewExpand returns a Provider, that expands all environment variables for a
// config.Map provided by the given Provider.
func NewExpand(base Provider) Provider {
	return &expandMapProvider{
		base: base,
	}
}

func (emp *expandMapProvider) Retrieve(ctx context.Context, onChange func(*ChangeEvent)) (Retrieved, error) {
	retr, err := emp.base.Retrieve(ctx, onChange)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve from base provider: %w", err)
	}
	cfgMap, err := retr.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get from base provider retrieved: %w", err)
	}
	for _, k := range cfgMap.AllKeys() {
		cfgMap.Set(k, expandStringValues(cfgMap.Get(k)))
	}
	return &simpleRetrieved{confMap: cfgMap}, nil
}

func (emp *expandMapProvider) Shutdown(ctx context.Context) error {
	return emp.base.Shutdown(ctx)
}

func expandStringValues(value interface{}) interface{} {
	switch v := value.(type) {
	default:
		return v
	case string:
		return expandEnv(v)
	case []interface{}:
		nslice := make([]interface{}, 0, len(v))
		for _, vint := range v {
			nslice = append(nslice, expandStringValues(vint))
		}
		return nslice
	case map[string]interface{}:
		nmap := make(map[interface{}]interface{}, len(v))
		for k, vint := range v {
			nmap[k] = expandStringValues(vint)
		}
		return nmap
	case map[interface{}]interface{}:
		nmap := make(map[interface{}]interface{}, len(v))
		for k, vint := range v {
			nmap[k] = expandStringValues(vint)
		}
		return nmap
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
