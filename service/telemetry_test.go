// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestBuildTelAttrs(t *testing.T) {
	buildInfo := component.NewDefaultBuildInfo()

	// Check default config
	cfg := telemetry.Config{}
	telAttrs := buildTelAttrs(buildInfo, cfg)

	assert.Len(t, telAttrs, 3)
	assert.Equal(t, buildInfo.Command, telAttrs[semconv.AttributeServiceName])
	assert.Equal(t, buildInfo.Version, telAttrs[semconv.AttributeServiceVersion])

	_, exists := telAttrs[semconv.AttributeServiceInstanceID]
	assert.True(t, exists)

	// Check override by nil
	cfg = telemetry.Config{
		Resource: map[string]*string{
			semconv.AttributeServiceName:       nil,
			semconv.AttributeServiceVersion:    nil,
			semconv.AttributeServiceInstanceID: nil,
		},
	}
	telAttrs = buildTelAttrs(buildInfo, cfg)

	// Attributes should not exist since we nil-ified all.
	assert.Len(t, telAttrs, 0)

	// Check override values
	strPtr := func(v string) *string { return &v }
	cfg = telemetry.Config{
		Resource: map[string]*string{
			semconv.AttributeServiceName:       strPtr("a"),
			semconv.AttributeServiceVersion:    strPtr("b"),
			semconv.AttributeServiceInstanceID: strPtr("c"),
		},
	}
	telAttrs = buildTelAttrs(buildInfo, cfg)

	assert.Len(t, telAttrs, 3)
	assert.Equal(t, "a", telAttrs[semconv.AttributeServiceName])
	assert.Equal(t, "b", telAttrs[semconv.AttributeServiceVersion])
	assert.Equal(t, "c", telAttrs[semconv.AttributeServiceInstanceID])
}
