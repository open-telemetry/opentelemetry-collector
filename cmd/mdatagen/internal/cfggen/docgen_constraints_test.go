// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestCfgDocConstraints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{name: "none", yaml: "type: string", want: ""},
		{name: "bounds", yaml: "type: int\nminimum: 1\nmaximum: 65535", want: "min: 1, max: 65535"},
		{name: "exclusive bounds", yaml: "type: double\nexclusiveMinimum: 0\nexclusiveMaximum: 30", want: "exclusive min: 0, exclusive max: 30"},
		{name: "length", yaml: "type: string\nminLength: 1\nmaxLength: 255", want: "min length: 1, max length: 255"},
		{name: "pattern", yaml: "type: string\npattern: '^[a-z]+$'", want: "pattern: `^[a-z]+$`"},
		{
			name: "pattern with alternation is escaped for markdown tables",
			yaml: "type: string\npattern: '^(http|https)$'",
			want: "pattern: `^(http\\|https)$`",
		},
		{
			name: "all together",
			yaml: "type: string\nminLength: 2\nmaxLength: 8\npattern: '^a|b$'",
			want: "min length: 2, max length: 8, pattern: `^a\\|b$`",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var cfg ConfigMetadata
			require.NoError(t, yaml.Unmarshal([]byte(tt.yaml), &cfg))
			assert.Equal(t, tt.want, CfgDocConstraints(&cfg))
		})
	}
}

func TestCfgDocConstraints_Nil(t *testing.T) {
	t.Parallel()
	assert.Empty(t, CfgDocConstraints(nil))
}
