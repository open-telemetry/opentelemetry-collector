// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/persistentqueue"

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"

	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/collector/featuregate"
)

// PersistRequestContextFeatureGate controls whether request context should be preserved in the persistent queue.
var PersistRequestContextFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"exporter.PersistRequestContext",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.128.0"),
	featuregate.WithRegisterDescription("controls whether context should be stored alongside requests in the persistent queue"),
)

type Encoding[T any] interface {
	// Marshal is a function that can marshal a request and its context into bytes.
	Marshal(context.Context, T) ([]byte, error)

	// Unmarshal is a function that can unmarshal bytes into a request and its context.
	Unmarshal([]byte) (context.Context, T, error)
}

type RequestEncoding[T any] interface {
	// MarshalTo marshals a request into a preallocated byte slice.
	// The size of the byte slice must be at least MarshalSize(T) bytes.
	MarshalTo(T, []byte) (int, error)

	// MarshalSize returns the size of the marshaled request.
	MarshalSize(T) int

	// Unmarshal unmarshals bytes into a request.
	Unmarshal([]byte) (T, error)
}

// defaultEncoder provides an interface for marshaling and unmarshaling requests along with their context.
type defaultEncoder[T any] struct {
	requestEncoding RequestEncoding[T]
}

func NewEncoder[T any](requestEncoding RequestEncoding[T]) Encoding[T] {
	return defaultEncoder[T]{requestEncoding: requestEncoding}
}

// requestDataKey is the key used to store request data in bytesMap.
const requestDataKey = "req"

var tracePropagator = propagation.TraceContext{}

func (re defaultEncoder[T]) Marshal(ctx context.Context, req T) ([]byte, error) {
	reqSize := re.requestEncoding.MarshalSize(req)

	if !PersistRequestContextFeatureGate.IsEnabled() {
		b := make([]byte, reqSize)
		_, err := re.requestEncoding.MarshalTo(req, b)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		return b, nil
	}

	bm := newBytesMap()
	tracePropagator.Inject(ctx, &bytesMapCarrier{bytesMap: bm})
	reqBuf, err := bm.setEmptyBytes(requestDataKey, reqSize)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	_, err = re.requestEncoding.MarshalTo(req, reqBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return *bm, nil
}

func (re defaultEncoder[T]) Unmarshal(b []byte) (context.Context, T, error) {
	if !PersistRequestContextFeatureGate.IsEnabled() {
		req, err := re.requestEncoding.Unmarshal(b)
		return context.Background(), req, err
	}

	bm := bytesMapFromBytes(b)
	if bm == nil {
		// Fall back to unmarshalling of the request alone.
		// This can happen if the data persisted by the version that doesn't support the context unmarshaling.
		req, err := re.requestEncoding.Unmarshal(b)
		return context.Background(), req, err
	}
	ctx := tracePropagator.Extract(context.Background(), &bytesMapCarrier{bytesMap: bm})
	reqBuf, err := bm.get(requestDataKey)
	var req T
	if err != nil {
		return context.Background(), req, fmt.Errorf("failed to read serialized request data: %w", err)
	}
	req, err = re.requestEncoding.Unmarshal(reqBuf)
	return ctx, req, err
}

// bytesMap is a slice of bytes that represents a map-like structure for storing key-value pairs.
// It's optimized for efficient memory usage for low number of key-value pairs with big values.
// The format is a sequence of key-value pairs encoded as:
//   - 1 byte length of the key
//   - key bytes
//   - 4 byte length of the value
//   - value bytes
type bytesMap []byte

const (
	// prefix bytes to denote the bytesMap serialization: 0x00 magic byte + 0x01 version of the encoder.
	magicByte      = byte(0x00)
	formatV1Byte   = byte(0x01)
	prefixBytesLen = 2

	initialCapacity = 256
)

func newBytesMap() *bytesMap {
	bm := bytesMap(make([]byte, 0, initialCapacity))
	bm = append(bm, magicByte, formatV1Byte)
	return &bm
}

// setEmptyBytes sets the specified key in the map, reserves the given number of bytes for the value,
// and returns a byte slice for the value. Must be called only once for each key.
func (bm *bytesMap) setEmptyBytes(key string, size int) ([]byte, error) {
	if len(key) > math.MaxUint8 {
		return nil, errors.New("key param is too long")
	}
	if size < 0 || uint64(size) > math.MaxUint32 {
		return nil, fmt.Errorf("invalid value size %d", size)
	}

	*bm = append(*bm, byte(len(key)))
	*bm = append(*bm, key...)

	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(size)) //nolint:gosec // disable G115
	*bm = append(*bm, lenBuf[:]...)

	start := len(*bm)
	*bm = slices.Grow(*bm, size)
	*bm = []byte(*bm)[:start+size]

	return []byte(*bm)[start:], nil
}

// get scans sequentially for the first matching key and returns the value as bytes.
func (bm *bytesMap) get(k string) ([]byte, error) {
	for i := prefixBytesLen; i < len(*bm); {
		kl := int([]byte(*bm)[i])
		i++

		if i+kl > len(*bm) {
			return nil, io.ErrUnexpectedEOF
		}
		key := string([]byte(*bm)[i : i+kl])
		i += kl

		if i+4 > len(*bm) {
			return nil, io.ErrUnexpectedEOF
		}
		vLen := binary.LittleEndian.Uint32([]byte(*bm)[i:])
		i += 4

		if i+int(vLen) > len(*bm) {
			return nil, io.ErrUnexpectedEOF
		}
		val := []byte(*bm)[i : i+int(vLen)]
		i += int(vLen)

		if key == k {
			return val, nil
		}
	}
	return nil, nil
}

// keys returns header names in encounter order.
func (bm *bytesMap) keys() []string {
	var out []string
	for i := prefixBytesLen; i < len(*bm); {
		kl := int([]byte(*bm)[i])
		i++

		if i+kl > len(*bm) {
			break // malformed entry
		}
		out = append(out, string([]byte(*bm)[i:i+kl]))
		i += kl

		if i+4 > len(*bm) {
			break // malformed entry
		}
		vLen := binary.LittleEndian.Uint32([]byte(*bm)[i:])
		i += 4 + int(vLen)
	}
	return out
}

func bytesMapFromBytes(b []byte) *bytesMap {
	if len(b) < prefixBytesLen || b[0] != magicByte || b[1] != formatV1Byte {
		return nil
	}
	return (*bytesMap)(&b)
}

// bytesMapCarrier implements propagation.TextMapCarrier on top of bytesMap.
type bytesMapCarrier struct {
	*bytesMap
}

var _ propagation.TextMapCarrier = (*bytesMapCarrier)(nil)

// Set appends a new string entry; if the key already exists it is left unchanged.
func (c *bytesMapCarrier) Set(k, v string) {
	buf, _ := c.setEmptyBytes(k, len(v))
	copy(buf, v)
}

// Get scans sequentially for the first matching key.
func (c *bytesMapCarrier) Get(k string) string {
	v, _ := c.get(k)
	return string(v)
}

// Keys returns header names in encounter order.
func (c *bytesMapCarrier) Keys() []string {
	return c.keys()
}
