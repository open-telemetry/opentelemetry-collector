// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	jsoniter "github.com/json-iterator/go"
)

func UnmarshalJSONIterEntityRef(ms EntityRef, iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "schemaUrl", "schema_url":
			ms.orig.SchemaUrl = iter.ReadString()
		case "type":
			ms.orig.Type = iter.ReadString()
		case "idKeys", "id_keys":
			UnmarshalJSONStreamStringSlice(NewStringSlice(&ms.orig.IdKeys, ms.state), iter)
		case "descriptionKeys", "description_keys":
			UnmarshalJSONStreamStringSlice(NewStringSlice(&ms.orig.DescriptionKeys, ms.state), iter)
		default:
			iter.Skip()
		}
		return true
	})
}
