// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	_, err := r.Register("foo", StageStable)
	assert.Error(t, err)
	assert.Error(t, r.Set("foo", true))
	r.MustRegister("foo", StageStable, WithRegisterToVersion("next"))
	assert.Error(t, r.Set("foo", false))

	assert.Error(t, r.Set("deprecated", true))
	_, err = r.Register("deprecated", StageDeprecated)
	assert.Error(t, err)
	assert.Error(t, r.Set("deprecated", true))
	r.MustRegister("deprecated", StageDeprecated, WithRegisterToVersion("next"))
	assert.Error(t, r.Set("deprecated", true))
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
			name:  "StageDeprecated Flag",
			id:    "test-gate",
			stage: StageDeprecated,
			opts: []RegisterOption{
				WithRegisterToVersion("next"),
			},
			enabled:   false,
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
			name:      "StageDeprecated gate missing removal version",
			id:        "test-gate",
			stage:     StageDeprecated,
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
