// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"go.opentelemetry.io/collector/pdata/internal/json"
)

func UnmarshalJSONStreamStringSlice(ms StringSlice, iter *json.Iterator) {
	iter.ReadArrayCB(func(iter *json.Iterator) bool {
		*ms.orig = append(*ms.orig, iter.ReadString())
		return true
	})
}
