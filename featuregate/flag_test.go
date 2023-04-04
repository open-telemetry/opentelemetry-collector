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
			expected:    map[string]bool{"alpha": false, "beta": true, "stable": true},
			expectedStr: "-alpha,beta,stable",
		},
		{
			name:        "simple enable alpha",
			input:       "alpha",
			expected:    map[string]bool{"alpha": true, "beta": true, "stable": true},
			expectedStr: "alpha,beta,stable",
		},
		{
			name:        "plus enable alpha",
			input:       "+alpha",
			expected:    map[string]bool{"alpha": true, "beta": true, "stable": true},
			expectedStr: "alpha,beta,stable",
		},
		{
			name:        "disabled beta",
			input:       "-beta",
			expected:    map[string]bool{"alpha": false, "beta": false, "stable": true},
			expectedStr: "-alpha,-beta,stable",
		},
		{
			name:        "multiple items",
			input:       "-beta,alpha",
			expected:    map[string]bool{"alpha": true, "beta": false, "stable": true},
			expectedStr: "alpha,-beta,stable",
		},
		{
			name:        "multiple items with plus",
			input:       "-beta,+alpha",
			expected:    map[string]bool{"alpha": true, "beta": false, "stable": true},
			expectedStr: "alpha,-beta,stable",
		},
		{
			name:        "repeated items",
			input:       "alpha,-beta,-alpha",
			expected:    map[string]bool{"alpha": false, "beta": false, "stable": true},
			expectedStr: "-alpha,-beta,stable",
		},
		{
			name:        "multiple plus items",
			input:       "+alpha,+beta",
			expected:    map[string]bool{"alpha": true, "beta": true, "stable": true},
			expectedStr: "alpha,beta,stable",
		},
		{
			name:           "enable stable",
			input:          "stable",
			expectedSetErr: true,
			expected:       map[string]bool{"alpha": false, "beta": true, "stable": true},
			expectedStr:    "-alpha,beta,stable",
		},
		{
			name:           "disable stable",
			input:          "stable",
			expectedSetErr: true,
			expected:       map[string]bool{"alpha": false, "beta": true, "stable": true},
			expectedStr:    "-alpha,beta,stable",
		},
		{
			name:           "enable missing",
			input:          "missing",
			expectedSetErr: true,
			expected:       map[string]bool{"alpha": false, "beta": true, "stable": true},
			expectedStr:    "-alpha,beta,stable",
		},
		{
			name:           "disable missing",
			input:          "missing",
			expectedSetErr: true,
			expected:       map[string]bool{"alpha": false, "beta": true, "stable": true},
			expectedStr:    "-alpha,beta,stable",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			reg.MustRegister("alpha", StageAlpha)
			reg.MustRegister("beta", StageBeta)
			reg.MustRegister("stable", StageStable, WithRegisterToVersion("1.0.0"))
			v := NewFlag(reg)
			if tt.expectedSetErr {
				require.Error(t, v.Set(tt.input))
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
