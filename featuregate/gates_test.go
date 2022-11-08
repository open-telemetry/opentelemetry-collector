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

func TestRegistry(t *testing.T) {
	r := Registry{gates: map[string]Gate{}}

	gate := Gate{
		ID:          "foo",
		Description: "Test Gate",
		Enabled:     true,
	}

	assert.Empty(t, r.List())
	assert.False(t, r.IsEnabled(gate.ID))

	assert.NoError(t, r.Register(gate))
	assert.Len(t, r.List(), 1)
	assert.True(t, r.IsEnabled(gate.ID))

	assert.NoError(t, r.Apply(map[string]bool{gate.ID: false}))
	assert.False(t, r.IsEnabled(gate.ID))

	assert.Error(t, r.Register(gate))
	assert.Panics(t, func() {
		r.MustRegister(gate)
	})
}

func TestRegistryWithErrorApply(t *testing.T) {
	r := Registry{gates: map[string]Gate{}}

	assert.NoError(t, r.Register(Gate{
		ID:          "foo",
		Description: "Test Gate",
		stage:       Alpha,
	}))
	assert.NoError(t, r.Register(Gate{
		ID:             "stable-foo",
		Description:    "Test Gate",
		stage:          Stable,
		removalVersion: "next",
	}))

	tests := []struct {
		name        string
		gate        string
		enabled     bool
		shouldError bool
	}{
		{
			name:        "existing_gate",
			gate:        "foo",
			enabled:     false,
			shouldError: false,
		},
		{
			name:        "none_existing_gate",
			gate:        "bar",
			enabled:     false,
			shouldError: true,
		},
		{
			name:        "stable gate modified",
			gate:        "stable-foo",
			enabled:     false,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldError {
				assert.Error(t, r.Apply(map[string]bool{tt.gate: tt.enabled}))
				return
			}
			assert.NoError(t, r.Apply(map[string]bool{tt.gate: tt.enabled}))
			assert.Equal(t, tt.enabled, r.IsEnabled(tt.gate))
		})
	}
}

func TestRegisterGateLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name      string
		id        string
		stage     Stage
		opts      []RegistryOption
		enabled   bool
		shouldErr bool
	}{
		{
			name:      "Alpha Flag",
			id:        "test-gate",
			stage:     Alpha,
			enabled:   false,
			shouldErr: false,
		},
		{
			name:  "Alpha Flag with all options",
			id:    "test-gate",
			stage: Alpha,
			opts: []RegistryOption{
				WithRegisterDescription("test-gate"),
				WithRegisterReferenceURL("http://example.com/issue/1"),
				WithRegisterRemovalVersion(""),
			},
			enabled:   false,
			shouldErr: false,
		},
		{
			name:      "Beta Flag",
			id:        "test-gate",
			stage:     Beta,
			enabled:   true,
			shouldErr: false,
		},
		{
			name:  "Stable Flag",
			id:    "test-gate",
			stage: Stable,
			opts: []RegistryOption{
				WithRegisterRemovalVersion("next"),
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
			name:      "Stable gate missing removal version",
			id:        "test-gate",
			stage:     Stable,
			shouldErr: true,
		},
		{
			name:      "Duplicate gate",
			id:        "existing-gate",
			stage:     Stable,
			shouldErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			require.NoError(t, r.RegisterID("existing-gate", Beta))
			if tc.shouldErr {
				assert.Error(t, r.RegisterID(tc.id, tc.stage, tc.opts...), "Must error when registering gate")
				assert.Panics(t, func() {
					r.MustRegisterID(tc.id, tc.stage, tc.opts...)
				})
				return
			}
			assert.NoError(t, r.RegisterID(tc.id, tc.stage, tc.opts...), "Must not error when registering feature gate")
			assert.Equal(t, tc.enabled, r.IsEnabled(tc.id), "Must match the expected enabled value")
		})
	}
}

func TestGateMethods(t *testing.T) {
	g := &Gate{
		ID:             "test",
		Description:    "test gate",
		Enabled:        false,
		stage:          Alpha,
		referenceURL:   "http://example.com",
		removalVersion: "v0.64.0",
	}

	assert.Equal(t, "test", g.GetID())
	assert.Equal(t, "test gate", g.GetDescription())
	assert.Equal(t, false, g.IsEnabled())
	assert.Equal(t, Alpha, g.Stage())
	assert.Equal(t, "http://example.com", g.ReferenceURL())
	assert.Equal(t, "v0.64.0", g.RemovalVersion())
}

func TestStageNames(t *testing.T) {
	for expected, s := range map[string]Stage{
		"Alpha":   Alpha,
		"Beta":    Beta,
		"Stable":  Stable,
		"unknown": Stage(-1),
	} {
		assert.Equal(t, expected, s.String())
	}
}
