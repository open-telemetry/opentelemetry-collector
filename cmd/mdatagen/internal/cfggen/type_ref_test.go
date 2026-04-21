// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoTypeRef_String(t *testing.T) {
	tests := []struct {
		name     string
		ref      GoTypeRef
		expected string
	}{
		{
			name:     "internal type (no import)",
			ref:      GoTypeRef{ImportPath: "", TypeName: "Target"},
			expected: "Target",
		},
		{
			name:     "external type with package qualifier",
			ref:      GoTypeRef{ImportPath: "go.opentelemetry.io/collector/config/confighttp", TypeName: "ClientConfig"},
			expected: "confighttp.ClientConfig",
		},
		{
			name:     "standard library type",
			ref:      GoTypeRef{ImportPath: "time", TypeName: "Duration"},
			expected: "time.Duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.ref.String())
		})
	}
}

func TestGoTypeRef_Qualifier(t *testing.T) {
	tests := []struct {
		name     string
		ref      GoTypeRef
		expected string
	}{
		{
			name:     "no import path",
			ref:      GoTypeRef{ImportPath: ""},
			expected: "",
		},
		{
			name:     "nested package",
			ref:      GoTypeRef{ImportPath: "go.opentelemetry.io/collector/config/confighttp"},
			expected: "confighttp",
		},
		{
			name:     "single segment",
			ref:      GoTypeRef{ImportPath: "time"},
			expected: "time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.ref.Qualifier())
		})
	}
}

func TestResolveGoTypeRef_Internal(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected GoTypeRef
	}{
		{
			name:     "simple snake_case name",
			ref:      "target",
			expected: GoTypeRef{ImportPath: "", TypeName: "Target"},
		},
		{
			name:     "multi-word snake_case",
			ref:      "my_custom_type",
			expected: GoTypeRef{ImportPath: "", TypeName: "MyCustomType"},
		},
		{
			name:     "already formatted",
			ref:      "MyType",
			expected: GoTypeRef{ImportPath: "", TypeName: "MyType"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveGoTypeRef(tt.ref, "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveGoTypeRef_External(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected GoTypeRef
	}{
		{
			name: "full module path",
			ref:  "go.opentelemetry.io/collector/config/confighttp.client_config",
			expected: GoTypeRef{
				ImportPath: "go.opentelemetry.io/collector/config/confighttp",
				TypeName:   "ClientConfig",
			},
		},
		{
			name: "with version suffix",
			ref:  "go.opentelemetry.io/collector/scraper/scraperhelper.controller_config@v0.146.0",
			expected: GoTypeRef{
				ImportPath: "go.opentelemetry.io/collector/scraper/scraperhelper",
				TypeName:   "ControllerConfig",
			},
		},
		{
			name: "github package",
			ref:  "github.com/example/pkg/subpkg.MyType",
			expected: GoTypeRef{
				ImportPath: "github.com/example/pkg/subpkg",
				TypeName:   "MyType",
			},
		},
		{
			name: "type name needs formatting",
			ref:  "github.com/example/pkg.my_type",
			expected: GoTypeRef{
				ImportPath: "github.com/example/pkg",
				TypeName:   "MyType",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveGoTypeRef(tt.ref, "", "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveGoTypeRef_LocalAbsolute(t *testing.T) {
	rootPkg := "go.opentelemetry.io/collector"

	tests := []struct {
		name     string
		ref      string
		expected GoTypeRef
	}{
		{
			name: "repo-relative path",
			ref:  "/config/confighttp.client_config",
			expected: GoTypeRef{
				ImportPath: "go.opentelemetry.io/collector/config/confighttp",
				TypeName:   "ClientConfig",
			},
		},
		{
			name: "nested path",
			ref:  "/scraper/scraperhelper.controller_config",
			expected: GoTypeRef{
				ImportPath: "go.opentelemetry.io/collector/scraper/scraperhelper",
				TypeName:   "ControllerConfig",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveGoTypeRef(tt.ref, rootPkg, "")
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveGoTypeRef_LocalAbsolute_DifferentRoot(t *testing.T) {
	rootPkg := "github.com/open-telemetry/opentelemetry-collector-contrib"

	ref := "/receiver/hostmetricsreceiver/internal.scraper_config"
	result, err := ResolveGoTypeRef(ref, rootPkg, "")
	require.NoError(t, err)
	require.Equal(t, GoTypeRef{
		ImportPath: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal",
		TypeName:   "ScraperConfig",
	}, result)
}

func TestResolveGoTypeRef_LocalRelative(t *testing.T) {
	compPkg := "go.opentelemetry.io/collector/scraper/scraperhelper"

	tests := []struct {
		name     string
		ref      string
		expected GoTypeRef
	}{
		{
			name: "relative to component",
			ref:  "./internal/metadata.metrics_builder",
			expected: GoTypeRef{
				ImportPath: "go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadata",
				TypeName:   "MetricsBuilder",
			},
		},
		{
			name: "parent directory reference",
			ref:  "../otherpackage.some_type",
			expected: GoTypeRef{
				ImportPath: "go.opentelemetry.io/collector/scraper/otherpackage",
				TypeName:   "SomeType",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveGoTypeRef(tt.ref, "", compPkg)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveGoTypeRef_Errors(t *testing.T) {
	tests := []struct {
		name string
		ref  string
	}{
		{
			name: "empty reference",
			ref:  "",
		},
		{
			name: "external missing type after dot",
			ref:  "github.com/example/pkg.",
		},
		{
			name: "local absolute missing type after dot",
			ref:  "/config/confighttp.",
		},
		{
			name: "local relative missing type after dot",
			ref:  "./internal/metadata.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveGoTypeRef(tt.ref, "root", "comp")
			require.Error(t, err)
		})
	}
}

func TestResolveGoTypeRef_VersionStripped(t *testing.T) {
	ref := "go.opentelemetry.io/collector/config/confighttp.client_config@v0.146.0"
	result, err := ResolveGoTypeRef(ref, "", "")
	require.NoError(t, err)
	require.Equal(t, "go.opentelemetry.io/collector/config/confighttp", result.ImportPath)
	require.Equal(t, "ClientConfig", result.TypeName)
}
