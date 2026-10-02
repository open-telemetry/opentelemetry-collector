// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"
import (
	"sync/atomic"
)

// State defines an ownership state of pmetric.Metrics, plog.Logs, ptrace.Traces or pprofile.Profiles.
type State struct {
	refs  atomic.Int32
	state uint32

	arenas   []*Arena
	ai       int
	wire     []byte
	heapRefs []any
}

const (
	defaultState          uint32 = 0
	stateReadOnlyBit             = uint32(1 << 0)
	statePipelineOwnedBit        = uint32(1 << 1)
)

func NewState() *State {
	st := &State{
		state: defaultState,
	}
	st.refs.Store(1)
	if useProtoArena() {
		st.arenas = []*Arena{getArena()}
	}
	return st
}

// RetainWire keeps the protobuf input buffer alive so string/[]byte fields may alias it.
func (st *State) RetainWire(buf []byte) {
	if st == nil || len(st.arenas) == 0 {
		return
	}
	st.wire = buf
}

// CloneAndRetainWire copies buf and retains the copy when arenas are attached.
func (st *State) CloneAndRetainWire(buf []byte) []byte {
	if st == nil || len(st.arenas) == 0 {
		return buf
	}
	owned := append([]byte(nil), buf...)
	st.wire = owned
	return owned
}

// ResetArena rewinds every arena on st so the next request starts at the first buffer.
func (st *State) ResetArena() {
	if st == nil {
		return
	}
	for _, a := range st.arenas {
		a.off = 0
	}
	st.ai = 0
	st.wire = nil
	st.heapRefs = st.heapRefs[:0]
}

// DropArena returns every arena on st to the pool.
func (st *State) DropArena() {
	if st == nil {
		return
	}
	for _, a := range st.arenas {
		a.off = 0
		arenaPool.Put(a)
	}
	st.arenas = nil
	st.ai = 0
	st.wire = nil
	st.heapRefs = nil
}

func (st *State) MarkReadOnly() {
	st.state |= stateReadOnlyBit
}

func (st *State) IsReadOnly() bool {
	return st.state&stateReadOnlyBit != 0
}

// AssertMutable panics if the state is not StateMutable.
func (st *State) AssertMutable() {
	if st.state&stateReadOnlyBit != 0 {
		panic("invalid access to shared data")
	}
}

// MarkPipelineOwned marks the data as owned by the pipeline, returns true if the data were
// previously not owned by the pipeline, otherwise false.
func (st *State) MarkPipelineOwned() bool {
	if st.state&statePipelineOwnedBit != 0 {
		return false
	}
	st.state |= statePipelineOwnedBit
	return true
}

// Ref add one to the count of active references.
func (st *State) Ref() {
	st.refs.Add(1)
}

// Unref returns true if reference count got to 0 which means no more active references,
// otherwise it returns false.
func (st *State) Unref() bool {
	refs := st.refs.Add(-1)
	switch {
	case refs > 0:
		return false
	case refs == 0:
		return true
	default:
		panic("Cannot unref freed data")
	}
}
