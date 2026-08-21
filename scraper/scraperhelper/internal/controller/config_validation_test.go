// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestControllerConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     ControllerConfig
		wantErr string
	}{
		{
			name: "default configuration",
			cfg:  NewDefaultControllerConfig(),
		},
		{
			name:    "zero value configuration",
			cfg:     ControllerConfig{},
			wantErr: `"collection_interval": requires positive value`,
		},
		{
			name: "invalid timeout",
			cfg: ControllerConfig{
				CollectionInterval: time.Minute,
				Timeout:            -time.Minute,
			},
			wantErr: `"timeout": requires positive value`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}
