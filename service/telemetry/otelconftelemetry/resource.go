// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry"
)

func createResource(
	_ context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	res := newResource(set, componentConfig.(*Config))
	pcommonRes := pcommon.NewResource()
	for _, keyValue := range res.Attributes() {
		key := string(keyValue.Key)
		pcommonRes.Attributes().PutStr(key, mustAttributeValueString(key, keyValue.Value))
	}
	return pcommonRes, nil
}

func newResource(set telemetry.Settings, cfg *Config) *sdkresource.Resource {
	return resource.New(set.BuildInfo, cfg.Resource)
}

func mustAttributeValueString(k string, v attribute.Value) string {
	if v.Type() != attribute.STRING {
		// We only support string-type resource attributes in the configuration.
		panic(fmt.Errorf("attribute %q: expected string, got %s", k, v.Type()))
	}
	return v.AsString()
}
