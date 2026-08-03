// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/proto"
)

func TestUnmarshalProtoAnyValueArrayValueRecursionLimit(t *testing.T) {
	dest := NewAnyValue()
	err := dest.UnmarshalProto(nestedArrayValueProto(proto.RecursionLimit/2 + 1))
	require.ErrorIs(t, err, proto.ErrRecursionDepth)
}

func TestUnmarshalProtoAnyValueKeyValueListRecursionLimit(t *testing.T) {
	dest := NewAnyValue()
	err := dest.UnmarshalProto(nestedKeyValueListProto(proto.RecursionLimit/3 + 1))
	require.ErrorIs(t, err, proto.ErrRecursionDepth)
}

func nestedArrayValueProto(depth int) []byte {
	var payload []byte
	for range depth {
		arrayValue := protoField(0x0a, payload)
		payload = protoField(0x2a, arrayValue)
	}
	return payload
}

func nestedKeyValueListProto(depth int) []byte {
	var payload []byte
	for range depth {
		keyValue := protoField(0x12, payload)
		keyValueList := protoField(0x0a, keyValue)
		payload = protoField(0x32, keyValueList)
	}
	return payload
}

func protoField(tag byte, payload []byte) []byte {
	field := make([]byte, 0, 1+proto.Sov(uint64(len(payload)))+len(payload))
	field = append(field, tag)
	field = appendRecursionVarint(field, uint64(len(payload)))
	field = append(field, payload...)
	return field
}

func appendRecursionVarint(buf []byte, v uint64) []byte {
	for v >= 1<<7 {
		buf = append(buf, uint8(v&0x7f|0x80))
		v >>= 7
	}
	return append(buf, uint8(v))
}
