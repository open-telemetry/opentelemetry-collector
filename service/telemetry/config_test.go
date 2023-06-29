// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		success  bool
		hostproc string
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
		{
			name: "metric telemetry with valid host proc",
			cfg: &Config{
				Metrics: MetricsConfig{
					Level:    configtelemetry.LevelBasic,
					Address:  "127.0.0.1:3333",
					HostProc: "/host/proc",
				},
			},
			success:  true,
			hostproc: "/host/proc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hostproc != "" {
				os.Setenv("HOST_PROC", tt.hostproc)
			} else {
				os.Unsetenv("HOST_PROC")
			}
			err := tt.cfg.Validate()
			if tt.success {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
