// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfgen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRef_PkgPath(t *testing.T) {
	ref := Ref{
		Namespace: "go.opentelemetry.io/collector",
		Path:      "scraper/scraperhelper",
		Type:      "controller_config",
	}

	result := ref.PkgPath()
	expected := "go.opentelemetry.io/collector/scraper/scraperhelper"
	require.Equal(t, expected, result)
}

func TestIsInNamespace(t *testing.T) {
	tests := []struct {
		name        string
		packagePath string
		expected    bool
	}{
		{
			name:        "collector namespace",
			packagePath: "go.opentelemetry.io/collector/scraper/scraperhelper",
			expected:    true,
		},
		{
			name:        "contrib namespace",
			packagePath: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver",
			expected:    true,
		},
		{
			name:        "unsupported namespace",
			packagePath: "github.com/example/custom-receiver",
			expected:    false,
		},
		{
			name:        "partial match not enough",
			packagePath: "go.opentelemetry.io/custom",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInNamespace(tt.packagePath)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNamespace(t *testing.T) {
	tests := []struct {
		name        string
		packagePath string
		expected    string
		wantErr     bool
	}{
		{
			name:        "collector namespace",
			packagePath: "go.opentelemetry.io/collector/scraper/scraperhelper",
			expected:    "go.opentelemetry.io/collector",
			wantErr:     false,
		},
		{
			name:        "contrib namespace",
			packagePath: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver",
			expected:    "github.com/open-telemetry/opentelemetry-collector-contrib",
			wantErr:     false,
		},
		{
			name:        "exact namespace match",
			packagePath: "go.opentelemetry.io/collector",
			expected:    "go.opentelemetry.io/collector",
			wantErr:     false,
		},
		{
			name:        "unsupported namespace",
			packagePath: "github.com/example/custom-receiver",
			expected:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getNamespace(tt.packagePath)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "namespace not supported")
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetRef(t *testing.T) {
	tests := []struct {
		name     string
		refPath  string
		expected *Ref
		wantErr  bool
		errMsg   string
	}{
		{
			name:    "valid collector reference",
			refPath: "go.opentelemetry.io/collector/scraper/scraperhelper.controller_config",
			expected: &Ref{
				Namespace: "go.opentelemetry.io/collector",
				Path:      "scraper/scraperhelper",
				Type:      "controller_config",
			},
			wantErr: false,
		},
		{
			name:    "valid contrib reference",
			refPath: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver.config",
			expected: &Ref{
				Namespace: "github.com/open-telemetry/opentelemetry-collector-contrib",
				Path:      "receiver/mysqlreceiver",
				Type:      "config",
			},
			wantErr: false,
		},
		{
			name:    "unsupported namespace",
			refPath: "github.com/example/custom-receiver.config",
			wantErr: true,
			errMsg:  "namespace not supported",
		},
		{
			name:    "missing type",
			refPath: "go.opentelemetry.io/collector/scraper/scraperhelper",
			wantErr: true,
			errMsg:  "invalid reference path",
		},
		{
			name:    "empty path",
			refPath: "",
			wantErr: true,
			errMsg:  "namespace not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getRef(tt.refPath)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetRefURL(t *testing.T) {
	tests := []struct {
		name     string
		ref      Ref
		version  string
		expected string
	}{
		{
			name: "collector reference",
			ref: Ref{
				Namespace: "go.opentelemetry.io/collector",
				Path:      "scraper/scraperhelper",
				Type:      "controller_config",
			},
			version:  "v1.0.0",
			expected: "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/v1.0.0/scraper/scraperhelper/config.yaml",
		},
		{
			name: "contrib reference",
			ref: Ref{
				Namespace: "github.com/open-telemetry/opentelemetry-collector-contrib",
				Path:      "receiver/mysqlreceiver",
				Type:      "config",
			},
			version:  "v0.95.0",
			expected: "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib/v0.95.0/receiver/mysqlreceiver/config.yaml",
		},
		{
			name: "main version",
			ref: Ref{
				Namespace: "go.opentelemetry.io/collector",
				Path:      "processor/batchprocessor",
				Type:      "config",
			},
			version:  "main",
			expected: "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/main/processor/batchprocessor/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRefURL(tt.ref, tt.version)
			require.Equal(t, tt.expected, result)
		})
	}
}
