// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetFlag(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedConfigs []string
		expectedErr     string
	}{
		{
			name:            "single set",
			args:            []string{"--set=processors.batch.timeout=2s"},
			expectedConfigs: []string{"yaml:processors::batch::timeout: 2s"},
		},
		{
			name:            "set and config",
			args:            []string{"--set=processors.batch.timeout=2s", "--config=file:testdata/otelcol-nop.yaml"},
			expectedConfigs: []string{"file:testdata/otelcol-nop.yaml", "yaml:processors::batch::timeout: 2s"},
		},
		{
			name:            "config and set",
			args:            []string{"--config=file:testdata/otelcol-nop.yaml", "--set=processors.batch.timeout=2s"},
			expectedConfigs: []string{"file:testdata/otelcol-nop.yaml", "yaml:processors::batch::timeout: 2s"},
		},
		{
			name:        "invalid set",
			args:        []string{"--set=processors.batch.timeout:2s"},
			expectedErr: `invalid value "processors.batch.timeout:2s" for flag -set: missing equal sign`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flgs := flags()
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
