// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Logs is the top-level struct that is propagated through the logs pipeline.
// Use NewLogs to create new instance, zero-initialized instance is not valid for use.
type Logs internal.Logs

func (ms Logs) getOrig() *otlpcollectorlog.ExportLogsServiceRequest {
	return internal.GetOrigLogs(internal.Logs(ms))
}

// NewLogs creates a new Logs struct.
func NewLogs() Logs {
	orig := &otlpcollectorlog.ExportLogsServiceRequest{}
	return Logs(internal.NewLogs(orig, pcommon.StateExclusive))
}

// MoveTo moves the Logs instance overriding the destination and
// resetting the current instance to its zero value.
// Deprecated: [1.0.0-rc5] The method can be replaced with a plain assignment.
func (ms Logs) MoveTo(dest Logs) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlpcollectorlog.ExportLogsServiceRequest{}
}

// CopyTo copies the Logs instance overriding the destination.
func (ms Logs) CopyTo(dest Logs) {
	ms.ResourceLogs().CopyTo(dest.MutableResourceLogs())
}

// LogRecordCount calculates the total number of log records.
func (ms Logs) LogRecordCount() int {
	logCount := 0
	rss := ms.ResourceLogs()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ill := rs.ScopeLogs()
		for i := 0; i < ill.Len(); i++ {
			logs := ill.At(i)
			logCount += logs.LogRecords().Len()
		}
	}
	return logCount
}

// ResourceLogs returns the ResourceLogsSlice associated with this Logs.
func (ms Logs) ResourceLogs() ResourceLogsSlice {
	return newImmutableResourceLogsSlice(&ms.getOrig().ResourceLogs)
}

// MutableResourceLogs returns the MutableResourceLogsSlice associated with this Logs object.
// This method should be called at once per ConsumeLogs call if the slice has to be changed,
// otherwise use ResourceLogs method.
func (ms Logs) MutableResourceLogs() MutableResourceLogsSlice {
	if internal.GetLogsState(internal.Logs(ms)) == pcommon.StateShared {
		rms := NewResourceLogsSlice()
		ms.ResourceLogs().CopyTo(rms)
		ms.getOrig().ResourceLogs = *rms.getOrig()
		internal.SetLogsState(internal.Logs(ms), pcommon.StateExclusive)
		return rms
	}
	return newMutableResourceLogsSlice(&ms.getOrig().ResourceLogs)
}
