// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func marshalResource(res pcommon.Resource, buf *dataBuffer) {
	buf.logAttributes("Resource attributes", res.Attributes())
	buf.logResourceEntities(res.Entities())
}
