// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/pdata/internal/proto"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"unsafe"
)

// WireType represents the proto wire type.
type WireType int8

const (
	WireTypeVarint     WireType = 0
	WireTypeI64        WireType = 1
	WireTypeLen        WireType = 2
	WireTypeStartGroup WireType = 3
	WireTypeEndGroup   WireType = 4
	WireTypeI32        WireType = 5
)

var (
	ErrInvalidLength        = errors.New("proto: negative length found during unmarshaling")
	ErrIntOverflow          = errors.New("proto: integer overflow")
	ErrUnexpectedEndOfGroup = errors.New("proto: unexpected end of group")
)

// BytesToString converts data to a string. If unsafeUnmarshal is true, the
// returned string aliases data and the caller must keep data alive and
// immutable for as long as the string is used.
func BytesToString(data []byte, unsafeUnmarshal bool) string {
	if !unsafeUnmarshal {
		return string(data)
	}
	return unsafe.String(unsafe.SliceData(data), len(data))
}

// BytesToBytes copies data, or returns data itself when unsafeUnmarshal is true.
// If unsafeUnmarshal is true, the caller must keep data alive and immutable for
// as long as the returned slice is used.
func BytesToBytes(data []byte, unsafeUnmarshal bool) []byte {
	if len(data) == 0 {
		return nil
	}
	if unsafeUnmarshal {
		return data
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out
}

// countFieldLimit is the largest remaining message we will scan to size a
// repeated field exactly. Larger messages grow exponentially instead; a full
// tag walk of a 10MB ScopeLogs is more expensive than a few slice reallocs.
const countFieldLimit = 4096

// GrowRepeated grows s for another repeated element. Small remaining messages
// are sized exactly; large ones double capacity to avoid a second proto scan.
func GrowRepeated[T any](s []T, buf []byte, pos int, fieldNum int32) []T {
	if cap(s) > len(s) {
		return s
	}
	extra := 8
	if remaining := len(buf) - pos; remaining > 0 && remaining <= countFieldLimit {
		extra = 1 + CountField(buf, pos, fieldNum)
	} else if cap(s) > extra {
		extra = cap(s)
	}
	return slices.Grow(s, extra)
}

// GrowCap grows s by extra capacity.
func GrowCap[T any](s []T, extra int) []T {
	return slices.Grow(s, extra)
}

// CountField counts remaining occurrences of fieldNum in buf starting at pos.
func CountField(buf []byte, pos int, fieldNum int32) int {
	n := 0
	for pos < len(buf) {
		num, wireType, next, err := ConsumeTag(buf, pos)
		if err != nil {
			return n
		}
		pos = next
		if num == fieldNum {
			n++
		}
		pos, err = ConsumeUnknown(buf, pos, wireType)
		if err != nil {
			return n
		}
	}
	return n
}

// ConsumeUnknown parses buf starting at pos as a wireType field, reporting the new position.
func ConsumeUnknown(buf []byte, pos int, wireType WireType) (int, error) {
	var err error
	l := len(buf)
	depth := 0
	for pos < l {
		switch wireType {
		case WireTypeVarint:
			_, pos, err = ConsumeVarint(buf, pos)
			return pos, err
		case WireTypeI64:
			_, pos, err = ConsumeI64(buf, pos)
			return pos, err
		case WireTypeLen:
			_, pos, err = ConsumeLen(buf, pos)
			return pos, err
		case WireTypeStartGroup:
			depth++
		case WireTypeEndGroup:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroup
			}
			depth--
		case WireTypeI32:
			_, pos, err = ConsumeI32(buf, pos)
			return pos, err
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}

		// Only when parsing a group can be here, if done return otherwise parse more tags.
		if depth == 0 {
			return pos, nil
		}

		// If in a group parsing, move to the next tag.
		_, wireType, pos, err = ConsumeTag(buf, pos)
		if err != nil {
			return 0, err
		}
	}
	return 0, io.ErrUnexpectedEOF
}

// ConsumeI64 parses buf starting at pos as a WireTypeI64 field, reporting the value and the new position.
func ConsumeI64(buf []byte, pos int) (uint64, int, error) {
	pos += 8
	if pos < 0 || pos > len(buf) {
		return 0, 0, io.ErrUnexpectedEOF
	}
	return binary.LittleEndian.Uint64(buf[pos-8:]), pos, nil
}

// ConsumeLen parses buf starting at pos as a WireTypeLen field, reporting the len and the new position.
func ConsumeLen(buf []byte, pos int) (int, int, error) {
	var num uint64
	var err error
	num, pos, err = ConsumeVarint(buf, pos)
	if err != nil {
		return 0, 0, err
	}
	length := int(num)
	if length < 0 {
		return 0, 0, ErrInvalidLength
	}
	pos += length
	if pos < 0 || pos > len(buf) {
		return 0, 0, io.ErrUnexpectedEOF
	}
	return length, pos, nil
}

// ConsumeI32 parses buf starting at pos as a WireTypeI32 field, reporting the value and the new position.
func ConsumeI32(buf []byte, pos int) (uint32, int, error) {
	pos += 4
	if pos < 0 || pos > len(buf) {
		return 0, 0, io.ErrUnexpectedEOF
	}
	return binary.LittleEndian.Uint32(buf[pos-4:]), pos, nil
}

// ConsumeTag parses buf starting at pos as a varint-encoded tag, reporting the new position.
func ConsumeTag(buf []byte, pos int) (int32, WireType, int, error) {
	tag, pos, err := ConsumeVarint(buf, pos)
	if err != nil {
		return 0, 0, 0, err
	}
	fieldNum := int32(tag >> 3)
	wireType := int8(tag & 0x7)
	if fieldNum <= 0 {
		return 0, 0, 0, fmt.Errorf("proto: Link: illegal field=%d (tag=%d, pos=%d)", fieldNum, tag, pos)
	}
	return fieldNum, WireType(wireType), pos, nil
}

// ConsumeVarint parses buf starting at pos as a varint-encoded uint64, reporting the new position.
func ConsumeVarint(buf []byte, pos int) (uint64, int, error) {
	l := len(buf)
	var num uint64
	for shift := uint(0); ; shift += 7 {
		if shift >= 64 {
			return 0, 0, ErrIntOverflow
		}
		if pos >= l {
			return 0, 0, io.ErrUnexpectedEOF
		}
		b := buf[pos]
		pos++
		num |= uint64(b&0x7F) << shift
		if b < 0x80 {
			break
		}
	}
	return num, pos, nil
}
