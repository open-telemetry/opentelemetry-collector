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

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		success bool
	}{
		{
			name: "basic metric telemetry",
			cfg: &Config{
				Metrics: MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: "127.0.0.1:3333",
				},
			},
			success: true,
		},
		{
			name: "invalid metric telemetry",
			cfg: &Config{
				Metrics: MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: "",
				},
			},
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.success {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
