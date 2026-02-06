// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	randomUUIDSpecialValue = "random-uuid"
)

var buildInfo = component.BuildInfo{
	Command: "otelcol",
	Version: "1.0.0",
}

func ptr[T any](v T) *T {
	return &v
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		want    map[string]string
		wantErr bool
	}{
		{
			name: "nil config defaults",
			cfg:  nil,
			want: map[string]string{
				"service.name":        "otelcol",
				"service.version":     "1.0.0",
				"service.instance.id": randomUUIDSpecialValue,
			},
		},
		{
			name: "empty config defaults",
			cfg:  &Config{},
			want: map[string]string{
				"service.name":        "otelcol",
				"service.version":     "1.0.0",
				"service.instance.id": randomUUIDSpecialValue,
			},
		},
		{
			name: "new format - override defaults",
			cfg: &Config{
				Attributes: []Attribute{
					{Name: "service.name", Value: "my-service"},
					{Name: "service.version", Value: "1.2.3"},
					{Name: "service.instance.id", Value: "123"},
				},
			},
			want: map[string]string{
				"service.name":        "my-service",
				"service.version":     "1.2.3",
				"service.instance.id": "123",
			},
		},
		{
			name: "new format - remove defaults",
			cfg: &Config{
				Attributes: []Attribute{
					{Name: "service.name", Value: nil},
					{Name: "service.version", Value: nil},
					{Name: "service.instance.id", Value: nil},
				},
			},
			want: map[string]string{},
		},
		{
			name: "new format - add custom",
			cfg: &Config{
				Attributes: []Attribute{
					{Name: "host.name", Value: "my-host"},
				},
			},
			want: map[string]string{
				"service.name":        "otelcol",
				"service.version":     "1.0.0",
				"service.instance.id": randomUUIDSpecialValue,
				"host.name":           "my-host",
			},
		},
		{
			name: "legacy format - override defaults",
			cfg: &Config{
				DeprecatedAttributes: map[string]any{
					"service.name":        "my-service",
					"service.version":     "1.2.3",
					"service.instance.id": "123",
				},
			},
			want: map[string]string{
				"service.name":        "my-service",
				"service.version":     "1.2.3",
				"service.instance.id": "123",
			},
		},
		{
			name: "legacy format - remove defaults",
			cfg: &Config{
				DeprecatedAttributes: map[string]any{
					"service.name":        nil,
					"service.version":     nil,
					"service.instance.id": nil,
				},
			},
			want: map[string]string{},
		},
		{
			name: "legacy format - add custom",
			cfg: &Config{
				DeprecatedAttributes: map[string]any{
					"host.name": "my-host",
				},
			},
			want: map[string]string{
				"service.name":        "otelcol",
				"service.version":     "1.0.0",
				"service.instance.id": randomUUIDSpecialValue,
				"host.name":           "my-host",
			},
		},
		{
			name: "new format overrides legacy",
			cfg: &Config{
				DeprecatedAttributes: map[string]any{
					"service.name": "legacy-name",
					"host.name":    "legacy-host",
				},
				Attributes: []Attribute{
					{Name: "service.name", Value: "new-name"},
				},
			},
			want: map[string]string{
				"service.name":        "new-name",
				"service.version":     "1.0.0",
				"service.instance.id": randomUUIDSpecialValue,
				"host.name":           "legacy-host",
			},
		},
		{
			name: "custom schema URL",
			cfg: &Config{
				SchemaURL: ptr("https://custom.schema/v1"),
			},
			want: map[string]string{
				"service.name":        "otelcol",
				"service.version":     "1.0.0",
				"service.instance.id": randomUUIDSpecialValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := New(buildInfo, tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			got := make(map[string]string)
			for _, attr := range res.Attributes() {
				got[string(attr.Key)] = attr.Value.Emit()
			}

			if tt.want["service.instance.id"] == randomUUIDSpecialValue {
				assert.Contains(t, got, "service.instance.id")
				_, err := uuid.Parse(got["service.instance.id"])
				require.NoError(t, err)
				delete(got, "service.instance.id")
				delete(tt.want, "service.instance.id")
			}

			assert.Equal(t, tt.want, got)

			// Check schema URL if specified
			if tt.cfg != nil && tt.cfg.SchemaURL != nil {
				assert.Equal(t, *tt.cfg.SchemaURL, res.SchemaURL())
			}
		})
	}
}

func TestNewErrors(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "new format - missing name",
			cfg: &Config{
				Attributes: []Attribute{
					{Name: "", Value: "value"},
				},
			},
		},
		{
			name: "new format - non-string value",
			cfg: &Config{
				Attributes: []Attribute{
					{Name: "key", Value: 123},
				},
			},
		},
		{
			name: "legacy format - non-string value",
			cfg: &Config{
				DeprecatedAttributes: map[string]any{
					"key": 123,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(buildInfo, tt.cfg)
			require.Error(t, err)
		})
	}
}

func TestLegacyPointerStringSupport(t *testing.T) {
	// Test *string pointer type in legacy attributes (for backward compatibility)
	cfg := &Config{
		DeprecatedAttributes: map[string]any{
			"service.name":        ptr("pointer-service"),
			"service.version":     ptr("pointer-version"),
			"service.instance.id": (*string)(nil), // nil pointer removes attribute
		},
	}

	res, err := New(buildInfo, cfg)
	require.NoError(t, err)

	got := make(map[string]string)
	for _, attr := range res.Attributes() {
		got[string(attr.Key)] = attr.Value.Emit()
	}

	assert.Equal(t, "pointer-service", got["service.name"])
	assert.Equal(t, "pointer-version", got["service.version"])
	assert.NotContains(t, got, "service.instance.id")
}

func TestInstanceIDGeneratorError(t *testing.T) {
	mockGenerator := func() (string, error) {
		return "", fmt.Errorf("failed to generate instance ID: %w", assert.AnError)
	}

	_, err := New(buildInfo, &Config{}, WithInstanceIDGenerator(mockGenerator))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate instance ID")
}

func pdataFromSdk(res *sdkresource.Resource) pcommon.Resource {
	pcommonRes := pcommon.NewResource()
	for _, keyValue := range res.Attributes() {
		pcommonRes.Attributes().PutStr(string(keyValue.Key), keyValue.Value.AsString())
	}
	return pcommonRes
}

func TestBuildResource(t *testing.T) {
	buildInfo := component.NewDefaultBuildInfo()

	// Check default config
	otelRes, err := New(buildInfo, nil)
	require.NoError(t, err)
	res := pdataFromSdk(otelRes)

	assert.Equal(t, 3, res.Attributes().Len())
	value, ok := res.Attributes().Get("service.name")
	assert.True(t, ok)
	assert.Equal(t, buildInfo.Command, value.AsString())
	value, ok = res.Attributes().Get("service.version")
	assert.True(t, ok)
	assert.Equal(t, buildInfo.Version, value.AsString())
	_, ok = res.Attributes().Get("service.instance.id")
	assert.True(t, ok)

	// Check override by nil using new format
	cfg := &Config{
		Attributes: []Attribute{
			{Name: "service.name", Value: nil},
			{Name: "service.version", Value: nil},
			{Name: "service.instance.id", Value: nil},
		},
	}
	otelRes, err = New(buildInfo, cfg)
	require.NoError(t, err)
	res = pdataFromSdk(otelRes)
	assert.Equal(t, 0, res.Attributes().Len())

	// Check override values using new format
	cfg = &Config{
		Attributes: []Attribute{
			{Name: "service.name", Value: "a"},
			{Name: "service.version", Value: "b"},
			{Name: "service.instance.id", Value: "c"},
		},
	}
	otelRes, err = New(buildInfo, cfg)
	require.NoError(t, err)
	res = pdataFromSdk(otelRes)

	assert.Equal(t, 3, res.Attributes().Len())
	value, ok = res.Attributes().Get("service.name")
	assert.True(t, ok)
	assert.Equal(t, "a", value.AsString())
	value, ok = res.Attributes().Get("service.version")
	assert.True(t, ok)
	assert.Equal(t, "b", value.AsString())
	value, ok = res.Attributes().Get("service.instance.id")
	assert.True(t, ok)
	assert.Equal(t, "c", value.AsString())
}
