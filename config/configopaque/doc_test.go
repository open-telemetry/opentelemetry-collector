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

package configopaque_test

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/config/configopaque"
)

func Example_opaqueString() {
	rawBytes := []byte(`{
		"Censored":   "sensitive",
		"Uncensored": "not sensitive"
	}`)

	// JSON unmarshaling
	var cfg ExampleConfigString
	err := json.Unmarshal(rawBytes, &cfg)
	if err != nil {
		panic(err)
	}

	// YAML marshaling
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("encoded cfg (YAML) is:\n%s\n\n", string(bytes))
	//Output: encoded cfg (YAML) is:
	// censored: '[REDACTED]'
	// uncensored: not sensitive
}

type ExampleConfigString struct {
	Censored   configopaque.String
	Uncensored string
}

func Example_opaqueSlice() {
	cfg := &ExampleConfigSlice{
		Censored:   []configopaque.String{"data", "is", "sensitive"},
		Uncensored: []string{"data", "is", "not", "sensitive"},
	}

	// JSON marshaling
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("encoded cfg (JSON) is\n%s\n\n", string(bytes))
	//Output: encoded cfg (JSON) is
	// {
	//   "Censored": [
	//     "[REDACTED]",
	//     "[REDACTED]",
	//     "[REDACTED]"
	//   ],
	//   "Uncensored": [
	//     "data",
	//     "is",
	//     "not",
	//     "sensitive"
	//   ]
	// }
}

type ExampleConfigSlice struct {
	Censored   []configopaque.String
	Uncensored []string
}

func Example_opaqueMap() {
	cfg := &ExampleConfigMap{
		Censored: map[string]configopaque.String{
			"token": "sensitivetoken",
		},
		Uncensored: map[string]string{
			"key":   "cloud.zone",
			"value": "zone-1",
		},
	}

	// yaml marshaling
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("encoded cfg (YAML) is:\n%s\n\n", string(bytes))
	//Output: encoded cfg (YAML) is:
	//censored:
	//     token: '[REDACTED]'
	//uncensored:
	//     key: cloud.zone
	//     value: zone-1
}

type ExampleConfigMap struct {
	Censored   map[string]configopaque.String
	Uncensored map[string]string
}
