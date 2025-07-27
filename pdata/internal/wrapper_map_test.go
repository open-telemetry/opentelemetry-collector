// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

func TestReadAttributeUnknownField(t *testing.T) {
	jsonStr := `[{"extra":""}]`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)
	var got []otlpcommon.KeyValue
	UnmarshalJSONIterMap(NewMap(&got, nil), iter)
	//  unknown fields should not be an error
	require.NoError(t, iter.Error())
	assert.Equal(t, []otlpcommon.KeyValue{{}}, got)
}

func TestReadAttributeValueUnknownField(t *testing.T) {
	// Key after value, to check that we correctly continue to process.
	jsonStr := `[{"value": {"unknown": {"extra":""}}, "key":"test"}]`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)
	var got []otlpcommon.KeyValue
	UnmarshalJSONIterMap(NewMap(&got, nil), iter)
	//  unknown fields should not be an error
	require.NoError(t, iter.Error())
	assert.Equal(t, []otlpcommon.KeyValue{{Key: "test"}}, got)
}
