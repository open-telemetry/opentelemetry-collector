// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestAttributes(t *testing.T) {
	tests := []struct {
		name           string
		cfg            telemetry.Config
		buildInfo      component.BuildInfo
		wantAttributes map[string]any
	}{
		{
			name:           "no build info and no resource config",
			cfg:            telemetry.Config{},
			wantAttributes: map[string]any{"service.name": "", "service.version": "", "service.instance.id": ""},
		},
		{
			name:           "build info and no resource config",
			cfg:            telemetry.Config{},
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			wantAttributes: map[string]any{"service.name": "otelcoltest", "service.version": "0.0.0-test", "service.instance.id": ""},
		},
		{
			name:           "no build info and resource config",
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": newPtr("resource.name"), "service.version": newPtr("resource.version"), "test": newPtr("test")}},
			wantAttributes: map[string]any{"service.name": "resource.name", "service.version": "resource.version", "test": "test", "service.instance.id": ""},
		},
		{
			name:           "build info and resource config",
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": newPtr("resource.name"), "service.version": newPtr("resource.version"), "test": newPtr("test")}},
			wantAttributes: map[string]any{"service.name": "resource.name", "service.version": "resource.version", "test": "test", "service.instance.id": ""},
		},
		{
			name:           "deleting a nil value",
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": nil, "service.version": newPtr("resource.version"), "test": newPtr("test")}},
			wantAttributes: map[string]any{"service.version": "resource.version", "test": "test", "service.instance.id": ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := attributes(resource.New(tt.buildInfo, tt.cfg.Resource), tt.cfg)
			require.Len(t, attrs, len(tt.wantAttributes))
			for k, v := range tt.wantAttributes {
				if k == "service.instance.id" {
					require.NotNil(t, attrs[k])
				} else {
					require.Equal(t, v, attrs[k])
				}
			}
		})
	}
}
