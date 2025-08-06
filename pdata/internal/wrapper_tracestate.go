// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"go.opentelemetry.io/collector/pdata/internal/json"
)

type TraceState struct {
	orig  *string
	state *State
}

func GetOrigTraceState(ms TraceState) *string {
	return ms.orig
}

func GetTraceStateState(ms TraceState) *State {
	return ms.state
}

func NewTraceState(orig *string, state *State) TraceState {
	return TraceState{orig: orig, state: state}
}

func GenerateTestTraceState() TraceState {
	var orig string
	FillOrigTestTraceState(&orig)
	state := StateMutable
	return NewTraceState(&orig, &state)
}

// MarshalJSONStreamTraceState marshals all properties from the current struct to the destination stream.
func MarshalJSONStreamTraceState(ms TraceState, dest *json.Stream) {
	dest.WriteString(*ms.orig)
}

// UnmarshalJSONIterTraceState marshals all properties from the current struct to the destination stream.
func UnmarshalJSONIterTraceState(ms TraceState, iter *json.Iterator) {
	*ms.orig = iter.ReadString()
}

func CopyOrigTraceState(dest, src *string) {
	*dest = *src
}

func FillOrigTestTraceState(orig *string) {
	*orig = "rojo=00f067aa0ba902b7"
}
