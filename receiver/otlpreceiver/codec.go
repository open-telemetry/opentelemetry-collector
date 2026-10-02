// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
)

type unsafeProtoDecoder interface {
	UnmarshalProtoUnsafe([]byte) error
}

type unsafeGRPCCodec struct {
	delegate encoding.CodecV2
}

func newUnsafeGRPCCodec() encoding.CodecV2 {
	return &unsafeGRPCCodec{delegate: encoding.GetCodecV2("proto")}
}

func (c *unsafeGRPCCodec) Marshal(v any) (mem.BufferSlice, error) {
	return c.delegate.Marshal(v)
}

func (c *unsafeGRPCCodec) Unmarshal(data mem.BufferSlice, v any) error {
	if decoder, ok := v.(unsafeProtoDecoder); ok {
		return decoder.UnmarshalProtoUnsafe(data.Materialize())
	}
	return c.delegate.Unmarshal(data, v)
}

func (c *unsafeGRPCCodec) Name() string {
	return c.delegate.Name()
}
