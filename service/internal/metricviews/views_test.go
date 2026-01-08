// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestDefaultViews(t *testing.T) {
	for _, tt := range []struct {
		name  string
		level configtelemetry.Level

		wantViewsCount int
	}{
		{
			name:           "None",
			level:          configtelemetry.LevelNone,
			wantViewsCount: 16,
		},
		{
			name:           "Basic",
			level:          configtelemetry.LevelBasic,
			wantViewsCount: 16,
		},
		{
			name:           "Normal",
			level:          configtelemetry.LevelNormal,
			wantViewsCount: 13,
		},
		{
			name:           "Detailed",
			level:          configtelemetry.LevelDetailed,
			wantViewsCount: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			views := DefaultViews(tt.level)
			assert.Len(t, views, tt.wantViewsCount)
		})
	}
}
