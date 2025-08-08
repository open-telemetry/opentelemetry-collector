// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/pdata/internal/proto"

func EncodeVarint(dAtA []byte, offset int, v uint64) int {
	offset -= Sov(v)
	base := offset
	for v >= 1<<7 {
		//nolint:gosec
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
