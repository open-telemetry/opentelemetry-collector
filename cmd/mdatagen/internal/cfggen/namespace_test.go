// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRef_Module(t *testing.T) {
	ref := NewRef("go.opentelemetry.io/collector/scraper/scraperhelper.controller_config", "")

	result := ref.Module()
	expected := "go.opentelemetry.io/collector/scraper/scraperhelper"
	require.Equal(t, expected, result)
}

func TestRef_HasNamespace(t *testing.T) {
	tests := []struct {
		name       string
		refPath    string
		hasNS      bool
		expectedNS string
	}{
		{
			name:       "collector namespace",
			refPath:    "go.opentelemetry.io/collector/scraper/scraperhelper.config",
			hasNS:      true,
			expectedNS: "go.opentelemetry.io/collector",
		},
		{
			name:       "contrib namespace",
			refPath:    "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver.config",
			hasNS:      true,
			expectedNS: "github.com/open-telemetry/opentelemetry-collector-contrib",
		},
		{
			name:       "unsupported namespace",
			refPath:    "github.com/example/custom-receiver.config",
			hasNS:      false,
			expectedNS: "",
		},
		{
			name:       "internal ref - no namespace",
			refPath:    "target_type",
			hasNS:      false,
			expectedNS: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := NewRef(tt.refPath, "")
			ns, ok := ref.Namespace()
			require.Equal(t, tt.hasNS, ok)
			require.Equal(t, tt.expectedNS, ns)
		})
	}
}

func TestRef_Namespace(t *testing.T) {
	tests := []struct {
		name     string
		refPath  string
		expected string
		hasNS    bool
	}{
		{
			name:     "collector namespace",
			refPath:  "go.opentelemetry.io/collector/scraper/scraperhelper.config",
			expected: "go.opentelemetry.io/collector",
			hasNS:    true,
		},
		{
			name:     "contrib namespace",
			refPath:  "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver.config",
			expected: "github.com/open-telemetry/opentelemetry-collector-contrib",
			hasNS:    true,
		},
		{
			name:     "unsupported namespace",
			refPath:  "github.com/example/custom-receiver.config",
			expected: "",
			hasNS:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := NewRef(tt.refPath, "")
			result, ok := ref.Namespace()
			require.Equal(t, tt.hasNS, ok)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRef_Validate(t *testing.T) {
	tests := []struct {
		name    string
		refPath string
		wantErr bool
	}{
		{
			name:    "valid collector reference",
			refPath: "go.opentelemetry.io/collector/scraper/scraperhelper.controller_config",
			wantErr: false,
		},
		{
			name:    "valid contrib reference",
			refPath: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver.config",
			wantErr: false,
		},
		{
			name:    "valid internal reference",
			refPath: "target_type",
			wantErr: false,
		},
		{
			name:    "valid local absolute reference",
			refPath: "/config/configauth.config",
			wantErr: false,
		},
		{
			name:    "empty path",
			refPath: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := NewRef(tt.refPath, "")
			err := ref.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRef_URL(t *testing.T) {
	tests := []struct {
		name     string
		refPath  string
		version  string
		expected string
	}{
		{
			name:     "collector reference",
			refPath:  "go.opentelemetry.io/collector/scraper/scraperhelper.controller_config",
			version:  "v1.0.0",
			expected: "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/v1.0.0/scraper/scraperhelper/config.schema.yaml",
		},
		{
			name:     "contrib reference",
			refPath:  "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver.config",
			version:  "v0.95.0",
			expected: "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib/v0.95.0/receiver/mysqlreceiver/config.schema.yaml",
		},
		{
			name:     "main version",
			refPath:  "go.opentelemetry.io/collector/processor/batchprocessor.config",
			version:  "main",
			expected: "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/main/processor/batchprocessor/config.schema.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := NewRef(tt.refPath, "")
			result, err := ref.URL(tt.version)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestNewRef_WithOrigin(t *testing.T) {
	tests := []struct {
		name     string
		refPath  string
		origin   string
		expected string
	}{
		{
			name:     "absolute ref resolves against namespace",
			refPath:  "/config/configauth.config",
			origin:   "go.opentelemetry.io/collector/config/confighttp",
			expected: "go.opentelemetry.io/collector/config/configauth.config",
		},
		{
			name:     "absolute ref with empty origin unchanged",
			refPath:  "/config/configauth.config",
			origin:   "",
			expected: "/config/configauth.config",
		},
		{
			name:     "relative ref resolves against module path",
			refPath:  "./internal/metadata.config",
			origin:   "go.opentelemetry.io/collector/config/confighttp",
			expected: "go.opentelemetry.io/collector/config/confighttp/internal/metadata.config",
		},
		{
			name:     "parent relative ref resolves against parent module",
			refPath:  "../configtls.tls_config",
			origin:   "go.opentelemetry.io/collector/config/confighttp",
			expected: "go.opentelemetry.io/collector/config/configtls.tls_config",
		},
		{
			name:     "external ref with origin unchanged",
			refPath:  "go.opentelemetry.io/collector/config/confighttp.client_config",
			origin:   "go.opentelemetry.io/collector/config/confighttp",
			expected: "go.opentelemetry.io/collector/config/confighttp.client_config",
		},
		{
			name:     "internal ref with origin unchanged",
			refPath:  "target_type",
			origin:   "go.opentelemetry.io/collector/config/confighttp",
			expected: "target_type",
		},
		{
			name:     "absolute ref inherits inline version from origin",
			refPath:  "/config/configauth.config",
			origin:   "go.opentelemetry.io/collector/config/confighttp@v1.2.0",
			expected: "go.opentelemetry.io/collector/config/configauth.config@v1.2.0",
		},
		{
			name:     "relative ref inherits inline version from origin",
			refPath:  "./internal/metadata.config",
			origin:   "go.opentelemetry.io/collector/config/confighttp@v1.2.0",
			expected: "go.opentelemetry.io/collector/config/confighttp/internal/metadata.config@v1.2.0",
		},
		{
			name:     "parent relative ref inherits inline version from origin",
			refPath:  "../configtls.tls_config",
			origin:   "go.opentelemetry.io/collector/config/confighttp@v1.2.0",
			expected: "go.opentelemetry.io/collector/config/configtls.tls_config@v1.2.0",
		},
		{
			name:     "no version propagated when origin has no version",
			refPath:  "/config/configauth.config",
			origin:   "go.opentelemetry.io/collector/config/confighttp",
			expected: "go.opentelemetry.io/collector/config/configauth.config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := NewRef(tt.refPath, tt.origin)
			require.Equal(t, tt.expected, ref.CacheKey())
		})
	}
}
