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

	assert.Empty(t, r.list())
	assert.False(t, r.isEnabled(gate.ID))

	assert.NoError(t, r.add(gate))
	assert.Len(t, r.list(), 1)
	assert.True(t, r.isEnabled(gate.ID))

	r.apply(map[string]bool{gate.ID: false})
	assert.False(t, r.isEnabled(gate.ID))

	assert.Error(t, r.add(gate))
}

func TestGlobalRegistry(t *testing.T) {
	resetForTest()
	gate := Gate{
		ID:          "foo",
		Description: "Test Gate",
		Enabled:     true,
	}

	assert.Empty(t, List())
	assert.False(t, IsEnabled(gate.ID))

	assert.NotPanics(t, func() { Register(gate) })
	assert.Len(t, List(), 1)
	assert.True(t, IsEnabled(gate.ID))

	Apply(map[string]bool{gate.ID: false})
	assert.False(t, IsEnabled(gate.ID))

	assert.Panics(t, func() { Register(gate) })
}

func resetForTest() *registry {
	reg.Lock()
	ret := reg
	reg = &registry{gates: make(map[string]Gate)}
	ret.Unlock()
	return ret
}
