// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nestedArrayBodyLogsJSON builds an OTLP/JSON Logs payload whose single log record body is an
// AnyValue nested `depth` arrayValue levels deep (AnyValue -> ArrayValue -> AnyValue -> ...).
func nestedArrayBodyLogsJSON(depth int) []byte {
	var b strings.Builder
	b.WriteString(`{"resourceLogs":[{"scopeLogs":[{"logRecords":[{"body":`)
	for range depth {
		b.WriteString(`{"arrayValue":{"values":[`)
	}
	b.WriteString(`{"stringValue":"x"}`)
	for range depth {
		b.WriteString(`]}}`)
	}
	b.WriteString(`}]}]}]}`)
	return []byte(b.String())
}

// nestedKvlistBodyLogsJSON builds an OTLP/JSON Logs payload whose single log record body is an
// AnyValue nested `depth` kvlistValue levels deep (AnyValue -> KeyValueList -> KeyValue -> ...).
func nestedKvlistBodyLogsJSON(depth int) []byte {
	var b strings.Builder
	b.WriteString(`{"resourceLogs":[{"scopeLogs":[{"logRecords":[{"body":`)
	for range depth {
		b.WriteString(`{"kvlistValue":{"values":[{"key":"k","value":`)
	}
	b.WriteString(`{"stringValue":"x"}`)
	for range depth {
		b.WriteString(`}]}}`)
	}
	b.WriteString(`}]}]}]}`)
	return []byte(b.String())
}

func TestUnmarshalJSONLogsRecursionLimit(t *testing.T) {
	// Well-formed, shallow nesting must still succeed for both value shapes.
	for _, buf := range [][]byte{nestedArrayBodyLogsJSON(10), nestedKvlistBodyLogsJSON(10)} {
		logs, err := (&JSONUnmarshaler{}).UnmarshalLogs(buf)
		require.NoError(t, err)
		assert.Equal(t, 1, logs.ResourceLogs().Len())
	}

	// Nesting far beyond the recursion limit must be rejected with an error rather than
	// accepted (and not left to overflow the goroutine stack at larger depths). Covers both
	// the arrayValue (AnyValue/ArrayValue) and kvlistValue (KeyValueList/KeyValue) chains.
	for _, buf := range [][]byte{nestedArrayBodyLogsJSON(1000), nestedKvlistBodyLogsJSON(1000)} {
		_, err := (&JSONUnmarshaler{}).UnmarshalLogs(buf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max recursion depth")
	}
}
