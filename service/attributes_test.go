// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"testing"

	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestAttributes(t *testing.T) {
	tests := []struct {
		name           string
		cfg            telemetry.Config
		buildInfo      component.BuildInfo
		wantAttributes []config.AttributeNameValue
	}{
		{
			name:           "no build info and no resource config",
			cfg:            telemetry.Config{},
			wantAttributes: []config.AttributeNameValue{{Name: "service.name", Value: ""}, {Name: "service.version", Value: ""}, {Name: "service.instance.id", Value: ""}},
		},
		{
			name:           "build info and no resource config",
			cfg:            telemetry.Config{},
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			wantAttributes: []config.AttributeNameValue{{Name: "service.name", Value: "otelcoltest"}, {Name: "service.version", Value: "0.0.0-test"}, {Name: "service.instance.id", Value: ""}},
		},
		{
			name:           "no build info and resource config",
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": ptr("resource.name"), "service.version": ptr("resource.version"), "test": ptr("test")}},
			wantAttributes: []config.AttributeNameValue{{Name: "service.name", Value: "resource.name"}, {Name: "service.version", Value: "resource.version"}, {Name: "test", Value: "test"}, {Name: "service.instance.id", Value: ""}},
		},
		{
			name:           "build info and resource config",
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": ptr("resource.name"), "service.version": ptr("resource.version"), "test": ptr("test")}},
			wantAttributes: []config.AttributeNameValue{{Name: "service.name", Value: "resource.name"}, {Name: "service.version", Value: "resource.version"}, {Name: "test", Value: "test"}, {Name: "service.instance.id", Value: ""}},
		},
		{
			name:           "deleting a nil value",
			buildInfo:      component.BuildInfo{Command: "otelcoltest", Version: "0.0.0-test"},
			cfg:            telemetry.Config{Resource: map[string]*string{"service.name": nil, "service.version": ptr("resource.version"), "test": ptr("test")}},
			wantAttributes: []config.AttributeNameValue{{Name: "service.version", Value: "resource.version"}, {Name: "test", Value: "test"}, {Name: "service.instance.id", Value: ""}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := attributes(resource.New(tt.buildInfo, tt.cfg.Resource), tt.cfg)
			require.Len(t, attrs, len(tt.wantAttributes))
			gotMap := map[string]any{}
			for _, v := range attrs {
				gotMap[v.Name] = v.Value
			}
			for _, v := range tt.wantAttributes {
				if v.Name == "service.instance.id" {
					require.NotNil(t, gotMap[v.Name])
				} else {
					require.Equal(t, v.Value, gotMap[v.Name])
				}
			}
		})
	}
}
