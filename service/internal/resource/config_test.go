// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
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
		name        string
		resourceCfg map[string]*string
		want        map[string]string
	}{
		{
			name:        "empty",
			resourceCfg: map[string]*string{},
			want: map[string]string{
				"service.name":        "otelcol",
				"service.version":     "1.0.0",
				"service.instance.id": randomUUIDSpecialValue,
			},
		},
		{
			name: "overwrite",
			resourceCfg: map[string]*string{
				"service.name":        ptr("my-service"),
				"service.version":     ptr("1.2.3"),
				"service.instance.id": ptr("123"),
			},
			want: map[string]string{
				"service.name":        "my-service",
				"service.version":     "1.2.3",
				"service.instance.id": "123",
			},
		},
		{
			name: "remove",
			resourceCfg: map[string]*string{
				"service.name":        nil,
				"service.version":     nil,
				"service.instance.id": nil,
			},
			want: map[string]string{},
		},
		{
			name: "add",
			resourceCfg: map[string]*string{
				"host.name": ptr("my-host"),
			},
			want: map[string]string{
				"service.name":        "otelcol",
				"service.version":     "1.0.0",
				"service.instance.id": randomUUIDSpecialValue,
				"host.name":           "my-host",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := New(buildInfo, tt.resourceCfg)
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
		})
	}
}

func pdataFromSdk(res *sdkresource.Resource) pcommon.Resource {
	// pcommon.NewResource is the best way to generate a new resource currently and is safe to use outside of tests.
	// Because the resource is signal agnostic, and we need a net new resource, not an existing one, this is the only
	// method of creating it without exposing internal packages.
	pcommonRes := pcommon.NewResource()
	for _, keyValue := range res.Attributes() {
		pcommonRes.Attributes().PutStr(string(keyValue.Key), keyValue.Value.AsString())
	}
	return pcommonRes
}

func TestBuildResource(t *testing.T) {
	defaultBuildInfo := component.NewDefaultBuildInfo()

	var resMap map[string]*string
	otelRes, err := New(defaultBuildInfo, resMap)
	require.NoError(t, err)
	res := pdataFromSdk(otelRes)

	assert.Equal(t, 3, res.Attributes().Len())
	value, ok := res.Attributes().Get("service.name")
	assert.True(t, ok)
	assert.Equal(t, defaultBuildInfo.Command, value.AsString())
	value, ok = res.Attributes().Get("service.version")
	assert.True(t, ok)
	assert.Equal(t, defaultBuildInfo.Version, value.AsString())

	_, ok = res.Attributes().Get("service.instance.id")
	assert.True(t, ok)

	resMap = map[string]*string{
		"service.name":        nil,
		"service.version":     nil,
		"service.instance.id": nil,
	}
	otelRes, err = New(buildInfo, resMap)
	require.NoError(t, err)
	res = pdataFromSdk(otelRes)

	assert.Equal(t, 0, res.Attributes().Len())

	strPtr := func(v string) *string { return &v }
	resMap = map[string]*string{
		"service.name":        strPtr("a"),
		"service.version":     strPtr("b"),
		"service.instance.id": strPtr("c"),
	}
	otelRes, err = New(buildInfo, resMap)
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

func TestDefaultAttributeValues(t *testing.T) {
	t.Run("defaults included", func(t *testing.T) {
		defaults, err := DefaultAttributeValues(buildInfo, nil)
		require.NoError(t, err)
		assert.Equal(t, buildInfo.Command, defaults["service.name"])
		assert.Equal(t, buildInfo.Version, defaults["service.version"])
		_, ok := defaults["service.instance.id"]
		assert.True(t, ok)
	})

	t.Run("defaults removed", func(t *testing.T) {
		removed := map[string]struct{}{
			"service.name":        {},
			"service.version":     {},
			"service.instance.id": {},
		}
		defaults, err := DefaultAttributeValues(buildInfo, removed)
		require.NoError(t, err)
		assert.NotContains(t, defaults, "service.name")
		assert.NotContains(t, defaults, "service.version")
		assert.NotContains(t, defaults, "service.instance.id")
	})

	t.Run("uuid failure", func(t *testing.T) {
		orig := newUUID
		t.Cleanup(func() { newUUID = orig })
		newUUID = func() (uuid.UUID, error) {
			return uuid.UUID{}, assert.AnError
		}

		_, err := DefaultAttributeValues(buildInfo, nil)
		require.ErrorContains(t, err, "failed to generate instance ID")
	})
}

func TestNewUUIDFailure(t *testing.T) {
	orig := newUUID
	t.Cleanup(func() { newUUID = orig })
	newUUID = func() (uuid.UUID, error) {
		return uuid.UUID{}, assert.AnError
	}

	_, err := New(buildInfo, map[string]*string{})
	require.ErrorContains(t, err, "failed to generate instance ID")
}
