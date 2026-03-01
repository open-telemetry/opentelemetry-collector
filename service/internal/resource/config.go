// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"go.opentelemetry.io/collector/component"
)

func New(buildInfo component.BuildInfo, resourceCfg map[string]*string) (*resource.Resource, error) {
	var telAttrs []attribute.KeyValue

	keys := make([]string, 0, len(resourceCfg))
	for k := range resourceCfg {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := resourceCfg[k]
		if v != nil {
			telAttrs = append(telAttrs, attribute.String(k, *v))
		}
	}

	removedDefaults := make(map[string]struct{})
	for _, key := range defaultAttributeKeys() {
		if v, ok := resourceCfg[key]; ok && v == nil {
			removedDefaults[key] = struct{}{}
		}
	}
	defaults, err := DefaultAttributeValues(buildInfo, removedDefaults)
	if err != nil {
		return nil, err
	}
	for _, key := range defaultAttributeKeys() {
		if _, ok := resourceCfg[key]; ok {
			continue
		}
		if value, ok := defaults[key]; ok {
			telAttrs = append(telAttrs, attribute.String(key, value))
		}
	}
	return resource.NewWithAttributes(semconv.SchemaURL, telAttrs...), nil
}

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

func defaultAttributeKeys() []string {
	return []string{
		string(semconv.ServiceNameKey),
		string(semconv.ServiceVersionKey),
		string(semconv.ServiceInstanceIDKey),
	}
}
