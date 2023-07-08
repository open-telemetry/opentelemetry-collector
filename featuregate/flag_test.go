// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlag(t *testing.T) {
	for _, tt := range []struct {
		name           string
		input          string
		expectedSetErr string
		expected       map[string]bool
		expectedStr    string
		strict         bool
	}{
		{
			name:        "empty item",
			input:       "",
			expected:    map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "-alpha,beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:        "simple enable alpha",
			input:       "alpha",
			expected:    map[string]bool{"alpha": true, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "alpha,beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:        "plus enable alpha",
			input:       "+alpha",
			expected:    map[string]bool{"alpha": true, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "alpha,beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:        "disabled beta",
			input:       "-beta",
			expected:    map[string]bool{"alpha": false, "beta": false, "deprecated": false, "stable": true},
			expectedStr: "-alpha,-beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:        "multiple items",
			input:       "-beta,alpha",
			expected:    map[string]bool{"alpha": true, "beta": false, "deprecated": false, "stable": true},
			expectedStr: "alpha,-beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:        "multiple items with plus",
			input:       "-beta,+alpha",
			expected:    map[string]bool{"alpha": true, "beta": false, "deprecated": false, "stable": true},
			expectedStr: "alpha,-beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:        "repeated items",
			input:       "alpha,-beta,-alpha",
			expected:    map[string]bool{"alpha": false, "beta": false, "deprecated": false, "stable": true},
			expectedStr: "-alpha,-beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:        "multiple plus items",
			input:       "+alpha,+beta",
			expected:    map[string]bool{"alpha": true, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "alpha,beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:        "enable stable",
			input:       "stable",
			expected:    map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "-alpha,beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:           "disable stable",
			input:          "-stable",
			expectedSetErr: "feature gate \"stable\" is stable, can not be disabled",
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
			strict:         false,
		},
		{
			name:           "enable deprecated",
			input:          "deprecated",
			expectedSetErr: "feature gate \"deprecated\" is deprecated, can not be enabled",
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
			strict:         false,
		},
		{
			name:        "disable deprecated",
			input:       "-deprecated",
			expected:    map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "-alpha,beta,-deprecated,stable",
			strict:      false,
		},
		{
			name:           "enable missing",
			input:          "missing",
			expectedSetErr: "missing",
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
			strict:         false,
		},
		{
			name:           "disable missing",
			input:          "missing",
			expectedSetErr: "missing",
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
			strict:         false,
		},
		{
			name:        "strict mode",
			input:       "alpha,beta,-alpha",
			expected:    map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr: "-alpha,beta,-deprecated,stable",
			strict:      true,
		},
		{
			name:           "strict mode but no alpha gate set",
			input:          "beta",
			expectedSetErr: "gate \"alpha\" is in Alpha and is not explicitly configured",
			expected:       map[string]bool{"alpha": false, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "-alpha,beta,-deprecated,stable",
			strict:         true,
		},
		{
			name:           "strict mode but no beta gate set",
			input:          "alpha",
			expectedSetErr: "gate \"beta\" is in Beta and is not explicitly configured",
			expected:       map[string]bool{"alpha": true, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "alpha,beta,-deprecated,stable",
			strict:         true,
		},
		{
			name:           "strict mode but beta gate disabled",
			input:          "alpha,-beta",
			expectedSetErr: "gate \"beta\" is in beta and must be explicitly enabled",
			expected:       map[string]bool{"alpha": true, "beta": false, "deprecated": false, "stable": true},
			expectedStr:    "alpha,-beta,-deprecated,stable",
			strict:         true,
		},
		{
			name:           "strict mode but stable gate set",
			input:          "alpha,beta,stable",
			expectedSetErr: "gate \"stable\" is stable and must not be configured",
			expected:       map[string]bool{"alpha": true, "beta": true, "deprecated": false, "stable": true},
			expectedStr:    "alpha,beta,-deprecated,stable",
			strict:         true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			reg.MustRegister("alpha", StageAlpha)
			reg.MustRegister("beta", StageBeta)
			reg.MustRegister("deprecated", StageDeprecated, WithRegisterToVersion("1.0.0"))
			reg.MustRegister("stable", StageStable, WithRegisterToVersion("1.0.0"))
			v := NewFlag(reg, &tt.strict)
			if tt.expectedSetErr != "" {
				require.ErrorContains(t, v.Set(tt.input), tt.expectedSetErr)
			} else {
				require.NoError(t, v.Set(tt.input))
			}
			got := map[string]bool{}
			reg.VisitAll(func(g *Gate) {
				got[g.ID()] = g.IsEnabled()
			})
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, tt.expectedStr, v.String())
		})
	}
}
