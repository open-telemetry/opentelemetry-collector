// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreportconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestConfigure(t *testing.T) {
	tests := []struct {
		name         string
		level        configtelemetry.Level
		wantViewsLen int
	}{
		{
			name:  "none",
			level: configtelemetry.LevelNone,
		},
		{
			name:         "basic",
			level:        configtelemetry.LevelBasic,
			wantViewsLen: 24,
		},
		{
			name:         "normal",
			level:        configtelemetry.LevelNormal,
			wantViewsLen: 24,
		},
		{
			name:         "detailed",
			level:        configtelemetry.LevelDetailed,
			wantViewsLen: 24,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, AllViews(tt.level), tt.wantViewsLen)
		})
	}
}
