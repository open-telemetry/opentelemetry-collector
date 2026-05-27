// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
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
		{
			name: "zero interval with controllers",
			cfg: ControllerConfig{
				Controllers: []component.ID{component.MustNewID("myext")},
			},
		},
		{
			name: "negative interval with controllers",
			cfg: ControllerConfig{
				CollectionInterval: -1,
				Controllers:        []component.ID{component.MustNewID("myext")},
			},
			wantErr: `"collection_interval": requires positive value`,
		},
		{
			name: "positive interval with controllers",
			cfg: ControllerConfig{
				CollectionInterval: time.Minute,
				Controllers:        []component.ID{component.MustNewID("myext")},
			},
		},
		{
			name: "duplicate controllers",
			cfg: ControllerConfig{
				CollectionInterval: time.Minute,
				Controllers: []component.ID{
					component.MustNewID("myext"),
					component.MustNewID("myext"),
					component.MustNewID("myext"),
				},
			},
			wantErr: `"controllers": duplicate extension ID "myext"`,
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
