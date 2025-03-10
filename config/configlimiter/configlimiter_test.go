// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configlimiter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/xextension/limiter"
)

var mockID = component.MustNewID("mock")
var otherID = component.MustNewID("other")

func TestGetLimiter(t *testing.T) {
	testCases := []struct {
		name     string
		cfg      Limitation
		limiter  extension.Extension
		expected error
	}{
		{
			name:     "obtain limiter",
			cfg:      Limitation{mockID},
			limiter:  limiter.NewNop(),
			expected: nil,
		},
		{
			name:     "wrong limiter",
			cfg:      Limitation{otherID},
			limiter:  limiter.NewNop(),
			expected: errNotLimiter,
		},
		{
			name:     "missing limiter",
			cfg:      Limitation{component.MustNewID("missing")},
			limiter:  limiter.NewNop(),
			expected: errLimiterNotFound,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ext := map[component.ID]component.Component{
				mockID:  tt.limiter,
				otherID: nil,
			}

			limExt, err := tt.cfg.GetLimiter(context.Background(), ext)

			// verify
			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
				assert.Nil(t, limExt)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, limExt)
			}
		})
	}
}
