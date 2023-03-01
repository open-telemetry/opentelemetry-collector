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

import (
	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
)

type Logs struct {
	*pLogs
}

type pLogs struct {
	orig  *otlpcollectorlog.ExportLogsServiceRequest
	state *State
}

func (ms Logs) AsShared() Logs {
	*ms.state = StateShared
	state := StateShared
	return Logs{&pLogs{orig: ms.orig, state: &state}}
}

func (ms Logs) GetState() *State {
	return ms.state
}

func (ms Logs) SetState(state *State) {
	ms.state = state
}

func (ms Logs) GetOrig() *otlpcollectorlog.ExportLogsServiceRequest {
	return ms.orig
}

func (ms Logs) SetOrig(orig *otlpcollectorlog.ExportLogsServiceRequest) {
	ms.orig = orig
}

func NewLogs(orig *otlpcollectorlog.ExportLogsServiceRequest) Logs {
	state := StateExclusive
	return Logs{&pLogs{orig: orig, state: &state}}
}

// LogsToProto internal helper to convert Logs to protobuf representation.
func LogsToProto(l Logs) otlplogs.LogsData {
	return otlplogs.LogsData{
		ResourceLogs: l.orig.ResourceLogs,
	}
}

// LogsFromProto internal helper to convert protobuf representation to Logs.
func LogsFromProto(orig otlplogs.LogsData) Logs {
	state := StateExclusive
	return Logs{&pLogs{
		orig: &otlpcollectorlog.ExportLogsServiceRequest{
			ResourceLogs: orig.ResourceLogs,
		},
		state: &state,
	}}
}
