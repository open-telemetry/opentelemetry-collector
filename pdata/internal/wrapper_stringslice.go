// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	jsoniter "github.com/json-iterator/go"
)

func UnmarshalJSONStreamStringSlice(ms StringSlice, iter *jsoniter.Iterator) {
	iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
		*ms.orig = append(*ms.orig, iter.ReadString())
		return true
	})
}
