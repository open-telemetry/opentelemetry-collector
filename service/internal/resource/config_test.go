// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
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
			res := New(buildInfo, tt.resourceCfg)
			got := make(map[string]string)
			for _, attr := range res.Attributes() {
				got[string(attr.Key)] = attr.Value.Emit()
			}

			if tt.want["service.instance.id"] == randomUUIDSpecialValue {
				assert.Contains(t, got, "service.instance.id")

				// Check that the value is a valid UUID.
				_, err := uuid.Parse(got["service.instance.id"])
				assert.NoError(t, err)

				// Remove so that we can compare the rest of the map.
				delete(got, "service.instance.id")
				delete(tt.want, "service.instance.id")
			}

			assert.EqualValues(t, tt.want, got)
		})
	}

}
