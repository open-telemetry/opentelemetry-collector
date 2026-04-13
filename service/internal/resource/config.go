// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"fmt"

	"github.com/google/uuid"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"go.opentelemetry.io/collector/component"
)

// newUUID is a variable to allow overriding UUID generation in tests.
var newUUID = uuid.NewRandom

func DefaultAttributeValues(buildInfo component.BuildInfo) (map[string]string, error) {
	instanceUUID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate instance ID: %w", err)
	}
	return map[string]string{
		string(semconv.ServiceNameKey):       buildInfo.Command,
		string(semconv.ServiceVersionKey):    buildInfo.Version,
		string(semconv.ServiceInstanceIDKey): instanceUUID.String(),
	}, nil
}
