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
	r.VisitAll(func(*Gate) {
		t.FailNow()
	})

	const id = "foo"
	g, err := r.Register(id, StageBeta, WithRegisterDescription("Test Gate"))
	require.NoError(t, err)
	r.VisitAll(func(gate *Gate) {
		assert.Equal(t, id, gate.ID())
	})
	assert.True(t, g.IsEnabled())

	require.NoError(t, r.Set(id, false))
	assert.False(t, g.IsEnabled())

	_, err = r.Register(id, StageBeta)
	require.ErrorIs(t, err, ErrAlreadyRegistered)
	assert.Panics(t, func() {
		r.MustRegister(id, StageBeta)
	})
}

func TestRegistryApplyError(t *testing.T) {
	r := NewRegistry()
	require.Error(t, r.Set("foo", true))
	r.MustRegister("bar", StageAlpha)

	require.Error(t, r.Set("foo", true))
	_, err := r.Register("foo", StageStable)
	require.Error(t, err)
	require.Error(t, r.Set("foo", true))
	r.MustRegister("foo", StageStable, WithRegisterToVersion("v1.0.0"))
	require.Error(t, r.Set("foo", false))

	require.Error(t, r.Set("deprecated", true))
	_, err = r.Register("deprecated", StageDeprecated)
	require.Error(t, err)
	require.Error(t, r.Set("deprecated", true))
	r.MustRegister("deprecated", StageDeprecated, WithRegisterToVersion("v1.0.0"))
	assert.Error(t, r.Set("deprecated", true))
}

func TestRegistryApply(t *testing.T) {
	r := NewRegistry()
	fooGate := r.MustRegister("foo", StageAlpha, WithRegisterDescription("Test Gate"))
	assert.False(t, fooGate.IsEnabled())
	require.NoError(t, r.Set(fooGate.ID(), true))
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
			id:        "test.gate",
			stage:     StageAlpha,
			enabled:   false,
			shouldErr: false,
		},
		{
			name:  "StageAlpha Flag with all options",
			id:    "test.gate",
			stage: StageAlpha,
			opts: []RegisterOption{
				WithRegisterDescription("test.gate"),
				WithRegisterReferenceURL("http://example.com/issue/1"),
				WithRegisterToVersion("v0.88.0"),
			},
			enabled:   false,
			shouldErr: false,
		},
		{
			name:      "StageBeta Flag",
			id:        "test.gate",
			stage:     StageBeta,
			enabled:   true,
			shouldErr: false,
		},
		{
			name:  "StageStable Flag",
			id:    "test.gate",
			stage: StageStable,
			opts: []RegisterOption{
				WithRegisterToVersion("v1.0.0-rcv.0014"),
			},
			enabled:   true,
			shouldErr: false,
		},
		{
			name:  "StageDeprecated Flag",
			id:    "test.gate",
			stage: StageDeprecated,
			opts: []RegisterOption{
				WithRegisterToVersion("v0.89.0"),
			},
			enabled:   false,
			shouldErr: false,
		},
		{
			name:      "Invalid stage",
			id:        "test.gate",
			stage:     Stage(-1),
			shouldErr: true,
		},
		{
			name:      "StageStable gate missing removal version",
			id:        "test.gate",
			stage:     StageStable,
			shouldErr: true,
		},
		{
			name:      "StageDeprecated gate missing removal version",
			id:        "test.gate",
			stage:     StageDeprecated,
			shouldErr: true,
		},
		{
			name:      "Duplicate gate",
			id:        "existing.gate",
			stage:     StageStable,
			shouldErr: true,
		},
		{
			name:      "Invalid gate name",
			id:        "+invalid.gate.name",
			stage:     StageAlpha,
			shouldErr: true,
		},
		{
			name:      "Invalid empty gate",
			id:        "",
			stage:     StageAlpha,
			shouldErr: true,
		},
		{
			name:      "Invalid gate to version",
			id:        "invalid.gate.to.version",
			stage:     StageAlpha,
			opts:      []RegisterOption{WithRegisterToVersion("invalid-version")},
			shouldErr: true,
		},
		{
			name:      "Invalid gate from version",
			id:        "invalid.gate.from.version",
			stage:     StageAlpha,
			opts:      []RegisterOption{WithRegisterFromVersion("invalid-version")},
			shouldErr: true,
		},
		{
			name:      "Invalid gate reference URL",
			id:        "invalid.gate.reference.URL",
			stage:     StageAlpha,
			opts:      []RegisterOption{WithRegisterReferenceURL(":invalid-url")},
			shouldErr: true,
		},
		{
			name:  "Empty version range",
			id:    "invalid.gate.version.range",
			stage: StageAlpha,
			opts: []RegisterOption{
				WithRegisterFromVersion("v0.88.0"),
				WithRegisterToVersion("v0.87.0"),
			},
			shouldErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			r.MustRegister("existing.gate", StageBeta)
			if tc.shouldErr {
				_, err := r.Register(tc.id, tc.stage, tc.opts...)
				require.Error(t, err)
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
