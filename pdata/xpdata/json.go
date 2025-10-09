// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xpdata // import "go.opentelemetry.io/collector/pdata/xpdata"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

type JSONMarshaler struct{}

func (*JSONMarshaler) MarshalAnyValue(value *otlpcommon.AnyValue) ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	internal.MarshalJSONOrigAnyValue(value, dest)
	if dest.Error() != nil {
		return nil, dest.Error()
	}
	return slices.Clone(dest.Buffer()), nil
}

type JSONUnmarshaler struct{}

func (*JSONUnmarshaler) UnmarshalAnyValue(buf []byte) (*otlpcommon.AnyValue, error) {
	iter := json.BorrowIterator(buf)
	defer json.ReturnIterator(iter)
	value := &otlpcommon.AnyValue{}
	internal.UnmarshalJSONOrigAnyValue(value, iter)
	if iter.Error() != nil {
		return nil, iter.Error()
	}
	return value, nil
}
