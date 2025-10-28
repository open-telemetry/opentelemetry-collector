// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xpdata // import "go.opentelemetry.io/collector/pdata/xpdata"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type JSONMarshaler struct{}

func (*JSONMarshaler) MarshalValue(value pcommon.Value) ([]byte, error) {
	av := internal.GetValueOrig(internal.ValueWrapper(value))
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	av.MarshalJSON(dest)
	if dest.Error() != nil {
		return nil, dest.Error()
	}
	return slices.Clone(dest.Buffer()), nil
}

type JSONUnmarshaler struct{}

func (*JSONUnmarshaler) UnmarshalValue(buf []byte) (pcommon.Value, error) {
	iter := json.BorrowIterator(buf)
	defer json.ReturnIterator(iter)
	value := &internal.AnyValue{}
	value.UnmarshalJSON(iter)
	if iter.Error() != nil {
		return pcommon.NewValueEmpty(), iter.Error()
	}
	return pcommon.Value(internal.NewValueWrapper(value, internal.NewState())), nil
}
