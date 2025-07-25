// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	jsoniter "github.com/json-iterator/go"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func UnmarshalJSONIterEntityRefSlice(ms EntityRefSlice, iter *jsoniter.Iterator) {
	iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
		*ms.orig = append(*ms.orig, &otlpcommon.EntityRef{})
		UnmarshalJSONIterEntityRef(NewEntityRef((*ms.orig)[len(*ms.orig)-1], ms.state), iter)
		return true
	})
}
