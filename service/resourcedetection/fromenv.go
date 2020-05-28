// Copyright The OpenTelemetry Authors
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

package resourcedetection

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Environment variable used by FromEnv to decode a resource.
const envVar = "OT_RESOURCE"

type FromEnv struct{}

// FromEnv is a detector that loads resource information from the OT_RESOURCE environment
// variable. A list of labels of the form `<key1>="<value1>",<key2>="<value2>",...` is
// accepted. Domain names and paths are accepted as label keys.
func (d *FromEnv) Detect(context.Context) (*resource.Resource, error) {
	labels := strings.TrimSpace(os.Getenv(envVar))
	if labels == "" {
		return resource.New(), nil
	}

	resLabels, err := decodeLabels(labels)
	if err != nil {
		return nil, err
	}

	return resource.New(resLabels...), nil
}

var labelRegex = regexp.MustCompile(`^\s*([[:ascii:]]{1,256}?)=("[[:ascii:]]{0,256}?")\s*,`)

func decodeLabels(s string) ([]kv.KeyValue, error) {
	kvs := []kv.KeyValue{}
	// Ensure a trailing comma, which allows us to keep the regex simpler
	s = strings.TrimRight(strings.TrimSpace(s), ",") + ","

	for len(s) > 0 {
		match := labelRegex.FindStringSubmatch(s)
		if len(match) == 0 {
			return nil, fmt.Errorf("invalid label formatting, remainder: %s", s)
		}
		v := match[2]
		if v == "" {
			v = match[3]
		} else {
			var err error
			if v, err = strconv.Unquote(v); err != nil {
				return nil, fmt.Errorf("invalid label formatting, remainder: %s, err: %s", s, err)
			}
		}
		kvs = append(kvs, kv.String(match[1], v))

		s = s[len(match[0]):]
	}
	return kvs, nil
}
