// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xpdata // import "go.opentelemetry.io/collector/pdata/xpdata"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type JSONMarshaler struct{}

func (*JSONMarshaler) MarshalValue(value pcommon.Value) ([]byte, error) {
	av := internal.GetOrigValue(internal.Value(value))
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	internal.MarshalJSONOrigAnyValue(av, dest)
	if dest.Error() != nil {
		return nil, dest.Error()
	}
	return slices.Clone(dest.Buffer()), nil
}

type JSONUnmarshaler struct{}

func (*JSONUnmarshaler) UnmarshalValue(buf []byte) (pcommon.Value, error) {
	iter := json.BorrowIterator(buf)
	defer json.ReturnIterator(iter)
	value := &otlpcommon.AnyValue{}
	internal.UnmarshalJSONOrigAnyValue(value, iter)
	if iter.Error() != nil {
		return pcommon.NewValueEmpty(), iter.Error()
	}
	return pcommon.Value(internal.NewValue(value, internal.NewState())), nil
}
