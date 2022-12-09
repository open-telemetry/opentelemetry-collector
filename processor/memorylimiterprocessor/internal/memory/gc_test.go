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

package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGCFunc(t *testing.T) {
	t.Parallel()

	called := false

	for _, tc := range []struct {
		name string
		fn   GCFunc
	}{
		{name: "default", fn: nil},
		{name: "override", fn: func() {
			called = true
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				tc.fn.Do()
			}, "Must not panic running GC function")
		})
	}
	assert.True(t, called, "Override function must have been called")
}
