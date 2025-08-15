// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

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
