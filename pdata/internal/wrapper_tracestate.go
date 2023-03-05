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
	parent Parent[*string]
}

type stubTraceStateParent struct {
	orig *string
}

func (vp stubTraceStateParent) EnsureMutability() {}

func (vp stubTraceStateParent) GetChildOrig() *string {
	return vp.orig
}

var _ Parent[*string] = (*stubTraceStateParent)(nil)

func (ms TraceState) GetOrig() *string {
	return ms.parent.GetChildOrig()
}

func (ms TraceState) EnsureMutability() {
	ms.parent.EnsureMutability()
}

func NewTraceStateFromOrig(orig *string) TraceState {
	return TraceState{parent: stubTraceStateParent{orig: orig}}
}

func NewTraceStateFromParent(parent Parent[*string]) TraceState {
	return TraceState{parent: parent}
}

func GenerateTestTraceState() TraceState {
	ms := NewTraceStateFromOrig(new(string))
	FillTestTraceState(ms)
	return ms
}

func FillTestTraceState(dest TraceState) {
	*dest.GetOrig() = "rojo=00f067aa0ba902b7"
}
