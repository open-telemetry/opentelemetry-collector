// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"go.opentelemetry.io/collector/featuregate"
)

var UseCustomProtoEncoding = featuregate.GlobalRegistry().MustRegister(
	"pdata.useCustomProtoEncoding",
	featuregate.StageBeta,
	featuregate.WithRegisterDescription("When enabled, enable custom proto encoding. This is required step to enable featuregate pdata.useProtoPooling."),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/issues/13631"),
	featuregate.WithRegisterFromVersion("v0.133.0"),
)

var UseProtoPooling = featuregate.GlobalRegistry().MustRegister(
	"pdata.useProtoPooling",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("When enabled, enable using local memory pools for underlying data that the pdata messages are pushed to."),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/issues/13631"),
	featuregate.WithRegisterFromVersion("v0.133.0"),
)

// State defines an ownership state of pmetric.Metrics, plog.Logs or ptrace.Traces.
type State int32

const (
	defaultState     State = 0
	stateReadOnlyBit       = State(1 << 0)
)

func NewState() *State {
	state := defaultState
	return &state
}

func (state *State) MarkReadOnly() {
	*state |= stateReadOnlyBit
}

func (state *State) IsReadOnly() bool {
	return *state&stateReadOnlyBit != 0
}

// AssertMutable panics if the state is not StateMutable.
func (state *State) AssertMutable() {
	if *state&stateReadOnlyBit != 0 {
		panic("invalid access to shared data")
	}
}
