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
				kind:      Internal,
			},
		},
		{
			name:    "internal ref",
			refPath: "target_type",
			expected: &Ref{
				namespace: "",
				schemaID:  "",
				defName:   "target_type",
				kind:      Internal,
			},
		},
		{
			name:    "local ref with absolute path",
			refPath: "/config/configauth.config",
			expected: &Ref{
				namespace: "",
				schemaID:  "/config/configauth",
				defName:   "config",
				kind:      Local,
			},
		},
		{
			name:    "local ref with relative path (./)",
			refPath: "./internal/metadata.config",
			expected: &Ref{
				namespace: "",
				schemaID:  "./internal/metadata",
				defName:   "config",
				kind:      Local,
			},
		},
		{
			name:    "local ref with relative path (../)",
			refPath: "../other.config",
			expected: &Ref{
				namespace: "",
				schemaID:  "../other",
				defName:   "config",
				kind:      Local,
			},
		},
		{
			name:    "local ref without def name",
			refPath: "/config/configauth",
			expected: &Ref{
				namespace: "",
				schemaID:  "/config/configauth",
				defName:   "",
				kind:      Local,
			},
		},
		{
			name:    "local ref empty",
			refPath: "../",
			expected: &Ref{
				namespace: "",
				schemaID:  "../",
				defName:   "",
				kind:      Local,
			},
		},
		{
			name:    "local ref empty short schemaId",
			refPath: "../.test",
			expected: &Ref{
				namespace: "",
				schemaID:  "../",
				defName:   "test",
				kind:      Local,
			},
		},
		{
			name:    "external ref without version",
			refPath: "go.opentelemetry.io/collector/config/confighttp.client_config",
			expected: &Ref{
				namespace: "go.opentelemetry.io/collector",
				schemaID:  "config/confighttp",
				defName:   "client_config",
				kind:      External,
			},
		},
		{
			name:    "external ref without def name",
			refPath: "go.opentelemetry.io/collector/config/confighttp",
			expected: &Ref{
				namespace: "go.opentelemetry.io/collector",
				schemaID:  "config/confighttp",
				defName:   "",
				kind:      External,
			},
		},
		{
			name:    "external ref without schema ID",
			refPath: "go.opentelemetry.io/collector",
			expected: &Ref{
				namespace: "go.opentelemetry.io/collector",
				schemaID:  "",
				defName:   "",
				kind:      External,
			},
		},
		{
			name:    "unknown namespace",
			refPath: "com.github.example/custom.xyz",
			expected: &Ref{
				namespace: "com.github.example",
				schemaID:  "custom",
				defName:   "xyz",
				kind:      External,
			},
		},
		{
			name:    "ref with wrong format",
			refPath: "some/path.with.dots",
			expected: &Ref{
				namespace: "some",
				schemaID:  "path",
				defName:   "with.dots",
				kind:      External,
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
			name:     "external ref with same-namespace origin joins schema IDs",
			refPath:  "go.opentelemetry.io/collector/config/confighttp.client_config",
			origin:   "go.opentelemetry.io/collector/config/confighttp",
			expected: "go.opentelemetry.io/collector/config/confighttp/config/confighttp.client_config",
		},
		{
			name:     "internal ref with origin unchanged",
			refPath:  "target_type",
			origin:   "go.opentelemetry.io/collector/config/confighttp",
			expected: "go.opentelemetry.io/collector/config/confighttp.target_type",
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
			name:    "missing schema ID with external ref",
			refPath: "go.opentelemetry.io/collector",
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
	ref := NewRef("go.opentelemetry.io/collector/config/confighttp.client_config")
	require.Equal(t, "go.opentelemetry.io/collector/config/confighttp.client_config", ref.String())
}

func TestRef_Module_EmptyNamespace(t *testing.T) {
	ref := NewRef("target_type")
	require.Empty(t, ref.Module())
}

func TestRef_URL_UnsupportedNamespace(t *testing.T) {
	ref := NewRef("unsupported.example.com/pkg/sub.config")
	_, err := ref.URL("v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported namespace")
}

func TestRef_Validate_LocalRefMissingSchemaID(t *testing.T) {
	ref := &Ref{schemaID: "", defName: "config", kind: Local}
	err := ref.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing schema ID in local reference")
}

func TestNamespaceOf_FallbackLastSlash(t *testing.T) {
	result := namespaceOf("com.example/some/path.type")
	require.Equal(t, "com.example/some", result)
}

func TestNamespaceOf_NoSlash(t *testing.T) {
	result := namespaceOf("noslash")
	require.Empty(t, result)
}
