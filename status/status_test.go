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

package status // import "go.opentelemetry.io/collector/receiver"

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	for i := Code(0); i < _maxCode; i++ {
		s := i.String()
		if s == "" || strings.Contains(s, "(") {
			t.Errorf("No valid string representation for code numbered %d", i)
		}
	}
}

func TestRoundtripJSON(t *testing.T) {
	for i := Code(0); i < _maxCode; i++ {
		b, err := json.Marshal(i)
		if err != nil {
			t.Errorf("Unable to marshal code %v: %v", i, err)
			continue
		}

		var c Code
		if err := json.Unmarshal(b, &c); err != nil {
			t.Errorf("Unable to unmarshal code from bytes %s: %v", string(b), err)
			continue
		}

		if i != c {
			t.Errorf("After roundtripping, %v != %v", i, c)
		}
	}
}
