// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"encoding/binary"
	"errors"
)

const (
	// field 1 << 3, wire type 5 (fixed32)
	protoTag1TypeByte = 0x0D

	// version of the request payload format set in the `format_version` field
	requestFormatVersion = uint32(1)
)

var ErrInvalidFormat = errors.New("invalid request payload format")

// isRequestPayloadV1 returns true if the given payload represents one of the wrappers around standard OpenTelemetry
// data types defined in internal/request.go and has the version set to 1.
func isRequestPayloadV1(data []byte) bool {
	if len(data) < 5 {
		return false
	}
	if data[0] != protoTag1TypeByte {
		return false
	}
	return binary.LittleEndian.Uint32(data[1:5]) == requestFormatVersion
}
