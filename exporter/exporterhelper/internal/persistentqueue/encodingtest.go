// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/persistentqueue"

import (
	"encoding/binary"
	"errors"
)

var ErrInvalidValue = errors.New("invalid value")

type Uint64Encoding struct{}

var _ RequestEncoding[uint64] = Uint64Encoding{}

func (e Uint64Encoding) MarshalTo(val uint64, buf []byte) (int, error) {
	binary.LittleEndian.PutUint64(buf, val)
	return e.MarshalSize(0), nil
}

func (Uint64Encoding) MarshalSize(uint64) int {
	return 8
}

func (Uint64Encoding) Unmarshal(bytes []byte) (uint64, error) {
	if len(bytes) != 8 {
		return 0, ErrInvalidValue
	}
	return binary.LittleEndian.Uint64(bytes), nil
}
