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

package featuregate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistry(t *testing.T) {
	r := registry{gates: map[string]Gate{}}

	gate := Gate{
		ID:          "foo",
		Description: "Test Gate",
		Enabled:     true,
	}

	assert.Empty(t, r.List())
	assert.False(t, r.IsEnabled(gate.ID))

	assert.NoError(t, r.add(gate))
	assert.Len(t, r.List(), 1)
	assert.True(t, r.IsEnabled(gate.ID))

	frozen := r.apply(map[string]bool{gate.ID: false})
	assert.False(t, frozen.IsEnabled(gate.ID))
	assert.True(t, r.IsEnabled(gate.ID))

	assert.Error(t, r.add(gate))
}

func BenchmarkFrozenRegistry_IsEnabled(b *testing.B) {
	frozen := registry{gates: map[string]Gate{"foo": {
		ID:          "foo",
		Description: "Test Gate",
		Enabled:     true,
	}}}

	for i := 0; i < b.N; i++ {
		assert.True(b, frozen.IsEnabled("foo"))
	}
}
