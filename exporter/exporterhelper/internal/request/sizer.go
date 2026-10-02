// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"

import (
	"encoding"
	"fmt"
)

// TODO: Move this back to queuebatch when remove the circular dependency.

var (
	_ encoding.TextMarshaler   = (*SizerType)(nil)
	_ encoding.TextUnmarshaler = (*SizerType)(nil)
)

type SizerType struct {
	val string
}

const (
	sizerTypeBytes    = "bytes"
	sizerTypeItems    = "items"
	sizerTypeRequests = "requests"
)

var (
	SizerTypeBytes    = SizerType{val: sizerTypeBytes}
	SizerTypeItems    = SizerType{val: sizerTypeItems}
	SizerTypeRequests = SizerType{val: sizerTypeRequests}
)

// UnmarshalText implements TextUnmarshaler interface.
func (s *SizerType) UnmarshalText(text []byte) error {
	switch str := string(text); str {
	case sizerTypeItems:
		*s = SizerTypeItems
	case sizerTypeBytes:
		*s = SizerTypeBytes
	case sizerTypeRequests:
		*s = SizerTypeRequests
	default:
		return fmt.Errorf("invalid sizer: %q", str)
	}
	return nil
}

func (s *SizerType) MarshalText() ([]byte, error) {
	return []byte(s.val), nil
}

func (s *SizerType) String() string {
	return s.val
}

// Sizer is an interface that returns the size of the given element.
type Sizer interface {
	Sizeof(Request) int64
}

func NewSizer(sizerType SizerType) Sizer {
	switch sizerType {
	case SizerTypeBytes:
		return NewBytesSizer()
	case SizerTypeItems:
		return NewItemsSizer()
	default:
		return RequestsSizer{}
	}
}

// RequestsSizer charges one request, or every child of an atomic request envelope.
type RequestsSizer struct{}

func (rs RequestsSizer) Sizeof(req Request) int64 {
	if counted, ok := req.(QueueRequestCount); ok {
		return counted.QueueRequestsCount()
	}
	return 1
}

type itemsSizer struct{}

func (itemsSizer) Sizeof(req Request) int64 {
	return int64(req.ItemsCount())
}

type bytesSizer struct{}

func (bytesSizer) Sizeof(req Request) int64 {
	return int64(req.BytesSize())
}

func NewItemsSizer() Sizer {
	return itemsSizer{}
}

func NewBytesSizer() Sizer {
	return bytesSizer{}
}
