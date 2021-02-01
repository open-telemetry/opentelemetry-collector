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

package viper

import (
	"fmt"
	"reflect"

	"github.com/alecthomas/units"
)

// Size is a human-friendly data size in bytes.
type Size int64

func humanStringToBytes(
	f reflect.Type,
	t reflect.Type,
	data interface{},
) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(Size(0)) {
		return data, nil
	}

	// Convert it by parsing.
	conv, err := units.ParseStrictBytes(data.(string))
	if err != nil {
		return data, fmt.Errorf("error converting %q to byte size: %v", data.(string), err)
	}

	return Size(conv), nil
}
