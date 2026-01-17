// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		name     string
		cfg      Config
		expected error
	}{
		{
			name:     "valid",
			cfg:      Config{component.MustNewID("nop")},
			expected: nil,
		},
		{
			name:     "duplicate-extension-reference",
			cfg:      Config{component.MustNewID("nop"), component.MustNewID("nop")},
			expected: errors.New(`references extension "nop" multiple times`),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expected != nil {
				require.ErrorContains(t, xconfmap.Validate(tt.cfg), tt.expected.Error())
			} else {
				require.NoError(t, xconfmap.Validate(tt.cfg))
			}
		})
	}
}
