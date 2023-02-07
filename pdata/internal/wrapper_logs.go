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
	sl *stateLogs
}

type stateLogs struct {
	orig  *otlpcollectorlog.ExportLogsServiceRequest
	state State
}

func GetLogsOrig(ms Logs) *otlpcollectorlog.ExportLogsServiceRequest {
	return ms.sl.orig
}

func GetLogsState(ms Logs) State {
	return ms.sl.state
}

// ResetStateLogs replaces the internal StateLogs with a new empty and the provided state.
func ResetStateLogs(ms Logs, s State) {
	ms.sl.orig = &otlpcollectorlog.ExportLogsServiceRequest{}
	ms.sl.state = s
}

func NewLogs(orig *otlpcollectorlog.ExportLogsServiceRequest, s State) Logs {
	return Logs{&stateLogs{orig: orig, state: s}}
}

// LogsToProto internal helper to convert Logs to protobuf representation.
func LogsToProto(l Logs) otlplogs.LogsData {
	return otlplogs.LogsData{
		ResourceLogs: l.sl.orig.ResourceLogs,
	}
}

// LogsFromProto internal helper to convert protobuf representation to Logs.
func LogsFromProto(orig otlplogs.LogsData) Logs {
	return Logs{&stateLogs{
		orig: &otlpcollectorlog.ExportLogsServiceRequest{
			ResourceLogs: orig.ResourceLogs,
		},
		state: StateExclusive,
	}}
}
