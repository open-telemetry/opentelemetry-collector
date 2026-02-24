// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRef(t *testing.T) {
	tests := []struct {
		name     string
		refPath  string
		expected *Ref
	}{
		{
			name:    "empty ref",
			refPath: "",
			expected: &Ref{
				namespace: "",
				schemaID:  "",
				defName:   "",
				version:   "",
			},
		},
		{
			name:    "internal ref",
			refPath: "target_type",
			expected: &Ref{
				namespace: "",
				schemaID:  "",
				defName:   "target_type",
				version:   "",
			},
		},
		{
			name:    "internal ref with version",
			refPath: "target_type@v1.0.0",
			expected: &Ref{
				namespace: "",
				schemaID:  "",
				defName:   "target_type",
				version:   "v1.0.0",
			},
		},
		{
			name:    "local ref with absolute path",
			refPath: "/config/configauth.config",
			expected: &Ref{
				namespace: "",
				schemaID:  "/config/configauth",
				defName:   "config",
				version:   "",
			},
		},
		{
			name:    "local ref with relative path (./)",
			refPath: "./internal/metadata.config",
			expected: &Ref{
				namespace: "",
				schemaID:  "./internal/metadata",
				defName:   "config",
				version:   "",
			},
		},
		{
			name:    "local ref with relative path (../)",
			refPath: "../other.config",
			expected: &Ref{
				namespace: "",
				schemaID:  "../other",
				defName:   "config",
				version:   "",
			},
		},
		{
			name:    "local ref without def name",
			refPath: "/config/configauth",
			expected: &Ref{
				namespace: "",
				schemaID:  "/config/configauth",
				defName:   "",
				version:   "",
			},
		},
		{
			name:    "local ref empty",
			refPath: "../",
			expected: &Ref{
				namespace: "",
				schemaID:  "../",
				defName:   "",
				version:   "",
			},
		},
		{
			name:    "local ref empty short schemaId",
			refPath: "../.test",
			expected: &Ref{
				namespace: "",
				schemaID:  "../",
				defName:   "test",
				version:   "",
			},
		},
		{
			name:    "local ref with version",
			refPath: "/config/configauth.config@v1.0.0",
			expected: &Ref{
				namespace: "",
				schemaID:  "/config/configauth",
				defName:   "config",
				version:   "v1.0.0",
			},
		},
		{
			name:    "external ref with version",
			refPath: "go.opentelemetry.io/collector/config/confighttp.client_config@v1.2.0",
			expected: &Ref{
				namespace: "go.opentelemetry.io/collector",
				schemaID:  "config/confighttp",
				defName:   "client_config",
				version:   "v1.2.0",
			},
		},
		{
			name:    "external ref without version",
			refPath: "go.opentelemetry.io/collector/config/confighttp.client_config",
			expected: &Ref{
				namespace: "go.opentelemetry.io/collector",
				schemaID:  "config/confighttp",
				defName:   "client_config",
				version:   "",
			},
		},
		{
			name:    "external ref without def name",
			refPath: "go.opentelemetry.io/collector/config/confighttp",
			expected: &Ref{
				namespace: "go.opentelemetry.io/collector",
				schemaID:  "config/confighttp",
				defName:   "",
				version:   "",
			},
		},
		{
			name:    "external ref without schema ID",
			refPath: "go.opentelemetry.io/collector",
			expected: &Ref{
				namespace: "go.opentelemetry.io/collector",
				schemaID:  "",
				defName:   "",
				version:   "",
			},
		},
		{
			name:    "unknown namespace",
			refPath: "com.github.example/custom.xyz@v1.0.0",
			expected: &Ref{
				namespace: "com.github.example",
				schemaID:  "custom",
				defName:   "xyz",
				version:   "v1.0.0",
			},
		},
		{
			name:    "ref with wrong format",
			refPath: "some/path.with.dots@v1.0.0",
			expected: &Ref{
				namespace: "some",
				schemaID:  "path",
				defName:   "with.dots",
				version:   "v1.0.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewRef(tt.refPath)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestWithOrigin(t *testing.T) {
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
			expected: "go.opentelemetry.io/collector/config/confighttp.target_type",
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
		{
			name:     "no version propagated when ref has own version",
			refPath:  "go.opentelemetry.io/collector/config/confighttp.client_config@v1.1.0",
			origin:   "go.opentelemetry.io/collector/config/confighttp.server_config@v1.2.0",
			expected: "go.opentelemetry.io/collector/config/confighttp.client_config@v1.1.0",
		},
		{
			name:     "version propagated when same namespace",
			refPath:  "go.opentelemetry.io/collector/config/confighttp.client_config",
			origin:   "go.opentelemetry.io/collector/config/confighttp.server_config@v1.2.0",
			expected: "go.opentelemetry.io/collector/config/confighttp.client_config@v1.2.0",
		},
		{
			name:     "no version propagated when different namespaces",
			refPath:  "go.opentelemetry.io/collector/config/confighttp.client_config",
			origin:   "github.com/open-telemetry/opentelemetry-collector-contrib/config/confighttp.server_config@v1.2.0",
			expected: "go.opentelemetry.io/collector/config/confighttp.client_config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := WithOrigin(tt.refPath, NewRef(tt.origin))
			require.Equal(t, tt.expected, ref.CacheKey())
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
		{
			name:    "missing def name",
			refPath: "/config/configauth",
			wantErr: true,
		},
		{
			name:    "version with local ref",
			refPath: "/config/configauth.config@v1.0.0",
			wantErr: true,
		},
		{
			name:    "version with internal ref",
			refPath: "target_type@v1.0.0",
			wantErr: true,
		},
		{
			name:    "missing schema ID with local ref",
			refPath: ".config@v1.0.0",
			wantErr: true,
		},
		{
			name:    "missing schema ID with external ref",
			refPath: "go.opentelemetry.io/collector@v1.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := NewRef(tt.refPath)
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
			ref := NewRef(tt.refPath)
			result, err := ref.URL(tt.version)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRef_String(t *testing.T) {
	ref := NewRef("go.opentelemetry.io/collector/config/confighttp.client_config@v1.2.0")
	require.Equal(t, "go.opentelemetry.io/collector/config/confighttp.client_config@v1.2.0", ref.String())
}
