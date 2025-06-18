// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2etest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNilToStringMap(t *testing.T) {
	tests := []struct {
		name        string
		expectedMap map[string]any
		expectedSub map[string]any
	}{
		{
			name: "subsection_null.yaml",
			expectedMap: map[string]any{
				"field": map[string]any{
					"key": nil,
				},
			},
			expectedSub: nil,
		},
		{
			name: "subsection_empty_map.yaml",
			expectedMap: map[string]any{
				"field": map[string]any{
					"key": map[string]any{},
				},
			},
			expectedSub: map[string]any{},
		},
		{
			name: "subsection_set_but_empty.yaml",
			expectedMap: map[string]any{
				"field": map[string]any{
					"key": nil,
				},
			},
			expectedSub: nil,
		},
		{
			name: "subsection_unset.yaml",
			expectedMap: map[string]any{
				"field": nil,
			},
			expectedSub: nil,
		},

		{
			name: "subsection_unset_empty_map.yaml",
			expectedMap: map[string]any{
				"field": map[string]any{},
			},
			expectedSub: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewResolver(t, tt.name)
			conf, err := resolver.Resolve(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.expectedMap, conf.ToStringMap())

			sub, err := conf.Sub("field::key")
			require.NoError(t, err)
			require.Equal(t, tt.expectedSub, sub.ToStringMap())
		})
	}
}

func TestNilToStringMapEnv(t *testing.T) {
	tests := []struct {
		envValue    string
		expectedMap map[string]any
		expectedSub map[string]any
	}{
		{
			envValue: "null",
			expectedMap: map[string]any{
				"field": map[string]any{
					"key": nil,
				},
			},
			expectedSub: nil,
		},
		{
			envValue: "{}",
			expectedMap: map[string]any{
				"field": map[string]any{
					"key": map[string]any{},
				},
			},
			expectedSub: map[string]any{},
		},
		{
			envValue: "",
			expectedMap: map[string]any{
				"field": map[string]any{
					"key": nil,
				},
			},
			expectedSub: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			t.Setenv("VALUE", tt.envValue)
			resolver := NewResolver(t, "types_map.yaml")
			conf, err := resolver.Resolve(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.expectedMap, conf.ToStringMap())
			sub, err := conf.Sub("field::key")
			require.NoError(t, err)
			require.Equal(t, tt.expectedSub, sub.ToStringMap())
		})
	}
}
