// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"fmt"

	"github.com/google/uuid"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"go.opentelemetry.io/collector/component"
)

var newUUID = uuid.NewRandom

func DefaultAttributeValues(buildInfo component.BuildInfo, removed map[string]struct{}) (map[string]string, error) {
	defaults := map[string]string{
		string(semconv.ServiceNameKey):    buildInfo.Command,
		string(semconv.ServiceVersionKey): buildInfo.Version,
	}
	if _, ok := removed[string(semconv.ServiceNameKey)]; ok {
		delete(defaults, string(semconv.ServiceNameKey))
	}
	if _, ok := removed[string(semconv.ServiceVersionKey)]; ok {
		delete(defaults, string(semconv.ServiceVersionKey))
	}
	if _, ok := removed[string(semconv.ServiceInstanceIDKey)]; !ok {
		instanceUUID, err := newUUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate instance ID: %w", err)
		}
		defaults[string(semconv.ServiceInstanceIDKey)] = instanceUUID.String()
	}
	return defaults, nil
}
