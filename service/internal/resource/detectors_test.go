// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func TestGetDetector(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		detectorName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "env detector",
			detectorName: "env",
			wantErr:      false,
		},
		{
			name:         "host detector",
			detectorName: "host",
			wantErr:      false,
		},
		{
			name:         "unknown detector",
			detectorName: "unknown",
			wantErr:      true,
			errContains:  "unknown detector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector, err := GetDetector(ctx, tt.detectorName)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, detector)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, detector)
			}
		})
	}
}

func TestGetDetectors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		detectorNames []string
		wantCount     int
		wantErr       bool
		errContains   string
	}{
		{
			name:          "empty list",
			detectorNames: []string{},
			wantCount:     0,
			wantErr:       false,
		},
		{
			name:          "nil list",
			detectorNames: nil,
			wantCount:     0,
			wantErr:       false,
		},
		{
			name:          "single detector",
			detectorNames: []string{"env"},
			wantCount:     1,
			wantErr:       false,
		},
		{
			name:          "multiple detectors",
			detectorNames: []string{"env", "host"},
			wantCount:     2,
			wantErr:       false,
		},
		{
			name:          "unknown detector in list",
			detectorNames: []string{"env", "unknown", "host"},
			wantCount:     0,
			wantErr:       true,
			errContains:   "unknown detector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detectors, err := GetDetectors(ctx, tt.detectorNames)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, detectors)
			} else {
				require.NoError(t, err)
				assert.Len(t, detectors, tt.wantCount)
			}
		})
	}
}

func TestEnvDetector(t *testing.T) {
	ctx := context.Background()

	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "key1=value1,key2=value2")
	t.Setenv("OTEL_SERVICE_NAME", "test-service")

	detector, err := newEnvDetector(ctx)
	require.NoError(t, err)
	require.NotNil(t, detector)

	res, err := detector.Detect(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)

	attrs := res.Attributes()
	var foundServiceName, foundKey1, foundKey2 bool
	for _, attr := range attrs {
		switch string(attr.Key) {
		case string(semconv.ServiceNameKey):
			assert.Equal(t, "test-service", attr.Value.AsString())
			foundServiceName = true
		case "key1":
			assert.Equal(t, "value1", attr.Value.AsString())
			foundKey1 = true
		case "key2":
			assert.Equal(t, "value2", attr.Value.AsString())
			foundKey2 = true
		}
	}

	assert.True(t, foundServiceName, "service.name should be set from OTEL_SERVICE_NAME")
	assert.True(t, foundKey1, "key1 should be set from OTEL_RESOURCE_ATTRIBUTES")
	assert.True(t, foundKey2, "key2 should be set from OTEL_RESOURCE_ATTRIBUTES")
}

func TestHostDetector(t *testing.T) {
	ctx := context.Background()

	detector, err := newHostDetector(ctx)
	require.NoError(t, err)
	require.NotNil(t, detector)

	res, err := detector.Detect(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)

	attrs := res.Attributes()
	assert.NotNil(t, attrs)
}

func TestEnvDetectorError(t *testing.T) {
	ctx := context.Background()

	detector := &envDetector{
		newResource: func(_ context.Context, _ ...resource.Option) (*resource.Resource, error) {
			return nil, assert.AnError
		},
	}

	res, err := detector.Detect(ctx)
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestHostDetectorError(t *testing.T) {
	ctx := context.Background()

	detector := &hostDetector{
		newResource: func(_ context.Context, _ ...resource.Option) (*resource.Resource, error) {
			return nil, assert.AnError
		},
	}

	res, err := detector.Detect(ctx)
	require.Error(t, err)
	assert.Nil(t, res)
}
