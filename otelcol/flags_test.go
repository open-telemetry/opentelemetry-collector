// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
)

func TestSetFlag(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedConfigs []string
		expectedErr     string
	}{
		{
			name:            "simple set",
			args:            []string{"--set=key=value"},
			expectedConfigs: []string{"yaml:key: value"},
		},
		{
			name:            "complex nested key",
			args:            []string{"--set=outer.inner=value"},
			expectedConfigs: []string{"yaml:outer::inner: value"},
		},
		{
			name:            "set array",
			args:            []string{"--set=key=[a, b, c]"},
			expectedConfigs: []string{"yaml:key: [a, b, c]"},
		},
		{
			name:            "set map",
			args:            []string{"--set=key={a: c}"},
			expectedConfigs: []string{"yaml:key: {a: c}"},
		},
		{
			name:            "set and config",
			args:            []string{"--set=key=value", "--config=file:testdata/otelcol-nop.yaml"},
			expectedConfigs: []string{"file:testdata/otelcol-nop.yaml", "yaml:key: value"},
		},
		{
			name:            "config and set",
			args:            []string{"--config=file:testdata/otelcol-nop.yaml", "--set=key=value"},
			expectedConfigs: []string{"file:testdata/otelcol-nop.yaml", "yaml:key: value"},
		},
		{
			name:        "invalid set",
			args:        []string{"--set=key:name"},
			expectedErr: `invalid value "key:name" for flag -set: missing equal sign`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flgs := flags(featuregate.NewRegistry())
			err := flgs.Parse(tt.args)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedConfigs, getConfigFlag(flgs))
		})
	}
}
