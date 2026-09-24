// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
)

// The recursion-depth guard lives on the four value types that can nest within themselves
// through the AnyValue attribute model. Rooting the deeply-nested payload at each of them
// exercises every guard's over-limit branch (integration coverage through the public
// JSONUnmarshaler exists in the signal packages, e.g. plog).

func nestedArrayValueJSON(depth int) []byte {
	var b strings.Builder
	b.WriteString(`{"values":[`)
	for range depth {
		b.WriteString(`{"arrayValue":{"values":[`)
	}
	b.WriteString(`{"stringValue":"x"}`)
	for range depth {
		b.WriteString(`]}}`)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

func nestedKeyValueListJSON(depth int) []byte {
	var b strings.Builder
	b.WriteString(`{"values":[{"key":"k","value":`)
	for range depth {
		b.WriteString(`{"kvlistValue":{"values":[{"key":"k","value":`)
	}
	b.WriteString(`{"stringValue":"x"}`)
	for range depth {
		b.WriteString(`}]}}`)
	}
	b.WriteString(`}]}`)
	return []byte(b.String())
}

func TestUnmarshalJSONArrayValueRecursionLimit(t *testing.T) {
	iter := json.BorrowIterator(nestedArrayValueJSON(200))
	defer json.ReturnIterator(iter)
	NewArrayValue().UnmarshalJSON(iter)
	require.Error(t, iter.Error())
	assert.Contains(t, iter.Error().Error(), "max recursion depth")
}

func TestUnmarshalJSONKeyValueListRecursionLimit(t *testing.T) {
	iter := json.BorrowIterator(nestedKeyValueListJSON(200))
	defer json.ReturnIterator(iter)
	NewKeyValueList().UnmarshalJSON(iter)
	require.Error(t, iter.Error())
	assert.Contains(t, iter.Error().Error(), "max recursion depth")
}
