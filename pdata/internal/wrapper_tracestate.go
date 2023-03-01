// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/pdata/internal"

type TraceState struct {
	*pTraceState
}

type pTraceState struct {
	orig   *string
	state  *State
	parent Parent[*string]
}

func (ms TraceState) GetOrig() *string {
	if *ms.state == StateDirty {
		ms.orig, ms.state = ms.parent.RefreshOrigState()
	}
	return ms.orig
}

func (ms TraceState) GetState() *string {
	return ms.orig
}

func (ms TraceState) EnsureMutability() {
	if *ms.state == StateShared {
		ms.parent.EnsureMutability()
	}
}

func NewTraceState(orig *string, parent Parent[*string]) TraceState {
	if parent == nil {
		state := StateExclusive
		return TraceState{&pTraceState{orig: orig, state: &state}}
	}
	return TraceState{&pTraceState{orig: orig, state: parent.GetState(), parent: parent}}
}

func GenerateTestTraceState() TraceState {
	ms := NewTraceState(new(string), nil)
	FillTestTraceState(ms)
	return ms
}

func FillTestTraceState(dest TraceState) {
	*dest.orig = "rojo=00f067aa0ba902b7"
}
