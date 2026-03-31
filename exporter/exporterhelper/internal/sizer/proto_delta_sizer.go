// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	math_bits "math/bits"
)

type protoDeltaSizer struct{}

// DeltaSize() returns the delta size of a proto slice when a new item is added.
// Example:
//
//	prevSize := proto1.Size()
//	proto1.RepeatedField().AppendEmpty() = proto2
//
// Then currSize of proto1 can be calculated as
//
//	currSize := (prevSize + sizer.DeltaSize(proto2.Size()))
//
// This is derived from:
// - opentelemetry-collector/pdata/internal/data/protogen/metrics/v1/metrics.pb.go
// - opentelemetry-collector/pdata/internal/data/protogen/logs/v1/logs.pb.go
// - opentelemetry-collector/pdata/internal/data/protogen/traces/v1/traces.pb.go
// - opentelemetry-collector/pdata/internal/data/protogen/profiles/v1development/profiles.pb.go
// which is generated with gogo/protobuf.
func (s *protoDeltaSizer) DeltaSize(newItemSize int) int {
	return 1 + newItemSize + sov(uint64(newItemSize))
}

func sov(x uint64) int {
	return (math_bits.Len64(x|1) + 6) / 7
}
