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
	"github.com/stretchr/testify/require"
)

func TestGlobalRegistry(t *testing.T) {
	assert.Same(t, globalRegistry, GlobalRegistry())
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	// Expect that no gates to visit.
	r.VisitAll(func(gate *Gate) {
		t.FailNow()
	})

	const id = "foo"
	g, err := r.Register(id, StageBeta, WithRegisterDescription("Test Gate"))
	assert.NoError(t, err)
	r.VisitAll(func(gate *Gate) {
		assert.Equal(t, id, gate.ID())
	})
	assert.True(t, g.IsEnabled())

	assert.NoError(t, r.Set(id, false))
	assert.False(t, g.IsEnabled())

	_, err = r.Register(id, StageBeta)
	assert.Error(t, err)
	assert.Panics(t, func() {
		r.MustRegister(id, StageBeta)
	})
}

func TestRegistryApplyError(t *testing.T) {
	r := NewRegistry()
	assert.Error(t, r.Set("foo", true))
	r.MustRegister("bar", StageAlpha)
	assert.Error(t, r.Set("foo", true))
	r.MustRegister("foo", StageStable, WithRegisterToVersion("next"))
	assert.Error(t, r.Set("foo", true))
}

func TestRegistryApply(t *testing.T) {
	r := NewRegistry()
	fooGate := r.MustRegister("foo", StageAlpha, WithRegisterDescription("Test Gate"))
	assert.False(t, fooGate.IsEnabled())
	assert.NoError(t, r.Set(fooGate.ID(), true))
	assert.True(t, fooGate.IsEnabled())
}

func TestRegisterGateLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name      string
		id        string
		stage     Stage
		opts      []RegisterOption
		enabled   bool
		shouldErr bool
	}{
		{
			name:      "StageAlpha Flag",
			id:        "test-gate",
			stage:     StageAlpha,
			enabled:   false,
			shouldErr: false,
		},
		{
			name:  "StageAlpha Flag with all options",
			id:    "test-gate",
			stage: StageAlpha,
			opts: []RegisterOption{
				WithRegisterDescription("test-gate"),
				WithRegisterReferenceURL("http://example.com/issue/1"),
				WithRegisterToVersion(""),
			},
			enabled:   false,
			shouldErr: false,
		},
		{
			name:      "StageBeta Flag",
			id:        "test-gate",
			stage:     StageBeta,
			enabled:   true,
			shouldErr: false,
		},
		{
			name:  "StageStable Flag",
			id:    "test-gate",
			stage: StageStable,
			opts: []RegisterOption{
				WithRegisterToVersion("next"),
			},
			enabled:   true,
			shouldErr: false,
		},
		{
			name:      "Invalid stage",
			id:        "test-gate",
			stage:     Stage(-1),
			shouldErr: true,
		},
		{
			name:      "StageStable gate missing removal version",
			id:        "test-gate",
			stage:     StageStable,
			shouldErr: true,
		},
		{
			name:      "Duplicate gate",
			id:        "existing-gate",
			stage:     StageStable,
			shouldErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			r.MustRegister("existing-gate", StageBeta)
			if tc.shouldErr {
				_, err := r.Register(tc.id, tc.stage, tc.opts...)
				assert.Error(t, err)
				assert.Panics(t, func() {
					r.MustRegister(tc.id, tc.stage, tc.opts...)
				})
				return
			}
			g, err := r.Register(tc.id, tc.stage, tc.opts...)
			require.NoError(t, err)
			assert.Equal(t, tc.enabled, g.IsEnabled())
		})
	}
}
