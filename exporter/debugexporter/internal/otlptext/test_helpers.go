// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/xpdata/entity"
)

func addEntityRefsToResource(resource pcommon.Resource) {
	entityRefs := entity.ResourceEntityRefs(resource)

	serviceRef := entityRefs.AppendEmpty()
	serviceRef.SetType("service")
	serviceRef.SetSchemaUrl("https://opentelemetry.io/schemas/1.21.0")
	serviceRef.IdKeys().Append("service.name")
	serviceRef.DescriptionKeys().Append("service.version")

	if _, exists := resource.Attributes().Get("host.name"); exists {
		hostRef := entityRefs.AppendEmpty()
		hostRef.SetType("host")
		hostRef.IdKeys().Append("host.name")
	}
}

func setupResourceWithEntityRefs(resource pcommon.Resource) {
	resource.Attributes().PutStr("service.name", "my-service")
	resource.Attributes().PutStr("service.version", "1.0.0")
	resource.Attributes().PutStr("host.name", "server-01")

	addEntityRefsToResource(resource)
}
