// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package json // import "go.opentelemetry.io/collector/pdata/internal/json"
import (
	jsoniter "github.com/json-iterator/go"
)

func BorrowIterator(data []byte) *jsoniter.Iterator {
	return jsoniter.ConfigFastest.BorrowIterator(data)
}

func ReturnIterator(s *jsoniter.Iterator) {
	jsoniter.ConfigFastest.ReturnIterator(s)
}
