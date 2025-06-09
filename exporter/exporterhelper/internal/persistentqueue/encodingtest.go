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

func (Uint64Encoding) Marshal(val uint64) ([]byte, error) {
	return binary.LittleEndian.AppendUint64([]byte{}, val), nil
}

func (Uint64Encoding) Unmarshal(bytes []byte) (uint64, error) {
	if len(bytes) != 8 {
		return 0, ErrInvalidValue
	}
	return binary.LittleEndian.Uint64(bytes), nil
}
