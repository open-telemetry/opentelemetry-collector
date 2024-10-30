// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestAttributes(t *testing.T) {
	tests := []struct {
		name           string
		cfg            telemetry.Config
		buildInfo      component.BuildInfo
		wantAttributes map[string]interface{}
	}{
		{
			name:           "no build info and no resource config",
			cfg:            telemetry.Config{},
			wantAttributes: map[string]interface{}{"service.name": "", "service.version": ""},
		},
		{
			name:           "build info and no resource config",
			cfg:            telemetry.Config{},
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			wantAttributes: map[string]interface{}{"service.name": "otelcoltest", "service.version": "0.0.0-test"},
		},
		{
			name:           "no build info and resource config",
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": newPtr("resource.name"), "service.version": newPtr("resource.version"), "test": newPtr("test")}},
			wantAttributes: map[string]interface{}{"service.name": "resource.name", "service.version": "resource.version", "test": "test"},
		},
		{
			name:           "build info and resource config",
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": newPtr("resource.name"), "service.version": newPtr("resource.version"), "test": newPtr("test")}},
			wantAttributes: map[string]interface{}{"service.name": "resource.name", "service.version": "resource.version", "test": "test"},
		},
		{
			name:           "deleting a nil value",
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": nil, "service.version": newPtr("resource.version"), "test": newPtr("test")}},
			wantAttributes: map[string]interface{}{"service.version": "resource.version", "test": "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := attributes(tt.buildInfo, tt.cfg)
			require.Equal(t, tt.wantAttributes, attrs)
		})
	}
}
