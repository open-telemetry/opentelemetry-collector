// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal/json"
)

func UnmarshalJSONIterInstrumentationScope(ms InstrumentationScope, iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "name":
			ms.orig.Name = iter.ReadString()
		case "version":
			ms.orig.Version = iter.ReadString()
		case "attributes":
			UnmarshalJSONIterMap(NewMap(&ms.orig.Attributes, ms.state), iter)
		case "droppedAttributesCount", "dropped_attributes_count":
			ms.orig.DroppedAttributesCount = json.ReadUint32(iter)
		default:
			iter.Skip()
		}
		return true
	})
}
