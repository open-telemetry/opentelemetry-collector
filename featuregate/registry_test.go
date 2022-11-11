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

	id := "foo"

	assert.Empty(t, r.List())
	assert.False(t, r.IsEnabled(id))

	assert.NoError(t, r.RegisterID(id, StageBeta, WithRegisterDescription("Test Gate")))
	assert.Len(t, r.List(), 1)
	assert.True(t, r.IsEnabled(id))

	assert.NoError(t, r.Apply(map[string]bool{id: false}))
	assert.False(t, r.IsEnabled(id))

	assert.Error(t, r.RegisterID(id, StageBeta))
	assert.Panics(t, func() {
		r.MustRegisterID(id, StageBeta)
	})
}

func TestRegistryWithErrorApply(t *testing.T) {
	r := Registry{gates: map[string]Gate{}}

	assert.NoError(t, r.RegisterID("foo", StageAlpha, WithRegisterDescription("Test Gate")))
	assert.NoError(t, r.RegisterID("stable-foo", StageStable, WithRegisterDescription("Test Gate"), WithRegisterRemovalVersion("next")))

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
			opts: []RegistryOption{
				WithRegisterDescription("test-gate"),
				WithRegisterReferenceURL("http://example.com/issue/1"),
				WithRegisterRemovalVersion(""),
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
			require.NoError(t, r.RegisterID("existing-gate", StageBeta))
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
