// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlag(t *testing.T) {
	for _, tt := range []struct {
		name           string
		input          string
		expectedSetErr bool
		expected       map[string]bool
		expectedStr    string
	}{
		{
			name:        "empty item",
			input:       "",
			expected:    map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "-alpha,beta,-deprecated,stable",
		},
		{
			name:        "simple enable alpha",
			input:       "alpha",
			expected:    map[string]bool{"alpha": true, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "alpha,beta,-deprecated,stable",
		},
		{
			name:        "plus enable alpha",
			input:       "+alpha",
			expected:    map[string]bool{"alpha": true, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "alpha,beta,-deprecated,stable",
		},
		{
			name:        "disabled beta",
			input:       "-beta",
			expected:    map[string]bool{"alpha": false, "beta": false, "deprecated": false, "stable": true},
			expectedStr: "-alpha,-beta,-deprecated,stable",
		},
		{
			name:        "multiple items",
			input:       "-beta,alpha",
			expected:    map[string]bool{"alpha": true, "beta": false, "deprecated": false, "stable": true},
			expectedStr: "alpha,-beta,-deprecated,stable",
		},
		{
			name:        "multiple items with plus",
			input:       "-beta,+alpha",
			expected:    map[string]bool{"alpha": true, "beta": false, "deprecated": false, "stable": true},
			expectedStr: "alpha,-beta,-deprecated,stable",
		},
		{
			name:        "repeated items",
			input:       "alpha,-beta,-alpha",
			expected:    map[string]bool{"alpha": false, "beta": false, "deprecated": false, "stable": true},
			expectedStr: "-alpha,-beta,-deprecated,stable",
		},
		{
			name:        "multiple plus items",
			input:       "+alpha,+beta",
			expected:    map[string]bool{"alpha": true, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "alpha,beta,-deprecated,stable",
		},
		{
			name:        "enable stable",
			input:       "stable",
			expected:    map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "-alpha,beta,-deprecated,stable",
		},
		{
			name:           "disable stable",
			input:          "-stable",
			expectedSetErr: true,
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
		},
		{
			name:           "enable deprecated",
			input:          "deprecated",
			expectedSetErr: true,
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
		},
		{
			name:        "disable deprecated",
			input:       "-deprecated",
			expected:    map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "-alpha,beta,-deprecated,stable",
		},
		{
			name:           "enable missing",
			input:          "missing",
			expectedSetErr: true,
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
		},
		{
			name:           "disable missing",
			input:          "missing",
			expectedSetErr: true,
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			reg.MustRegister("alpha", StageAlpha)
			reg.MustRegister("beta", StageBeta)
			reg.MustRegister("deprecated", StageDeprecated, WithRegisterToVersion("1.0.0"))
			reg.MustRegister("stable", StageStable, WithRegisterToVersion("1.0.0"))
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			reg.RegisterFlags(fs)
			registrationFlag := fs.Lookup(featureGatesFlag)
			require.NotNil(t, registrationFlag)
			if tt.expectedSetErr {
				require.Error(t, registrationFlag.Value.Set(tt.input))
			} else {
				require.NoError(t, registrationFlag.Value.Set(tt.input))
			}
			got := map[string]bool{}
			reg.VisitAll(func(g *Gate) {
				got[g.ID()] = g.IsEnabled()
			})
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, tt.expectedStr, registrationFlag.Value.String())
		})
	}
}

func TestFlagStringNotInitialize(t *testing.T) {
	flag := &flagValue{}
	assert.Empty(t, flag.String())
}
