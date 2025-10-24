// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

type MapWrapper struct {
	orig  *[]otlpcommon.KeyValue
	state *State
}

func GetMapOrig(ms MapWrapper) *[]otlpcommon.KeyValue {
	return ms.orig
}

func GetMapState(ms MapWrapper) *State {
	return ms.state
}

func NewMapWrapper(orig *[]otlpcommon.KeyValue, state *State) MapWrapper {
	return MapWrapper{orig: orig, state: state}
}

func GenTestMapWrapper() MapWrapper {
	orig := GenTestKeyValueSlice()
	return NewMapWrapper(&orig, NewState())
}
