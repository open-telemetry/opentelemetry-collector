// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/proto"
)

type Map struct {
	orig  *[]otlpcommon.KeyValue
	state *State
}

func GetOrigMap(ms Map) *[]otlpcommon.KeyValue {
	return ms.orig
}

func GetMapState(ms Map) *State {
	return ms.state
}

func NewMap(orig *[]otlpcommon.KeyValue, state *State) Map {
	return Map{orig: orig, state: state}
}

func GenerateTestMap() Map {
	orig := GenerateOrigTestKeyValueSlice()
	state := StateMutable
	return NewMap(&orig, &state)
}

func SizeProtoKeyValue(orig *otlpcommon.KeyValue) int {
	if orig == nil {
		return 0
	}
	var n int
	_ = n
	var l int
	_ = l
	l = len(orig.Key)
	if l > 0 {
		n += 1 + l + proto.Sov(uint64(l))
	}
	l = SizeProtoValue(&orig.Value)
	n += 1 + l + proto.Sov(uint64(l))
	return n
}
