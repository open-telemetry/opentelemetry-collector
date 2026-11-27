// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import "unicode/utf8"

// ValidateUTF8ExportLogsServiceRequest returns false when any string in req contains invalid UTF-8.
func ValidateUTF8ExportLogsServiceRequest(req *ExportLogsServiceRequest) bool {
	for _, rl := range req.ResourceLogs {
		if !validUTF8Resource(&rl.Resource) || !utf8.ValidString(rl.SchemaUrl) ||
			!validUTF8ScopeLogsSlice(rl.ScopeLogs) || !validUTF8ScopeLogsSlice(rl.DeprecatedScopeLogs) {
			return false
		}
	}
	return true
}

func validUTF8ScopeLogsSlice(scopes []*ScopeLogs) bool {
	for _, sl := range scopes {
		if !validUTF8InstrumentationScope(&sl.Scope) || !utf8.ValidString(sl.SchemaUrl) {
			return false
		}
		for _, lr := range sl.LogRecords {
			if !validUTF8LogRecord(lr) {
				return false
			}
		}
	}
	return true
}

func validUTF8LogRecord(lr *LogRecord) bool {
	return utf8.ValidString(lr.SeverityText) && utf8.ValidString(lr.EventName) &&
		validUTF8AnyValue(&lr.Body) && validUTF8KeyValueSlice(lr.Attributes)
}

// RejectInvalidUTF8ExportLogsServiceRequest removes log records containing invalid UTF-8 from req
// and returns the number removed.
func RejectInvalidUTF8ExportLogsServiceRequest(req *ExportLogsServiceRequest) int {
	rejected := 0
	req.ResourceLogs = removeIf(req.ResourceLogs, DeleteResourceLogs, func(rl *ResourceLogs) bool {
		if !validUTF8Resource(&rl.Resource) || !utf8.ValidString(rl.SchemaUrl) {
			for _, sl := range rl.ScopeLogs {
				rejected += len(sl.LogRecords)
			}
			return true
		}
		if len(rl.ScopeLogs) == 0 {
			return false
		}
		rl.ScopeLogs = removeIf(rl.ScopeLogs, DeleteScopeLogs, func(sl *ScopeLogs) bool {
			if !validUTF8InstrumentationScope(&sl.Scope) || !utf8.ValidString(sl.SchemaUrl) {
				rejected += len(sl.LogRecords)
				return true
			}
			if len(sl.LogRecords) == 0 {
				return false
			}
			sl.LogRecords = removeIf(sl.LogRecords, DeleteLogRecord, func(lr *LogRecord) bool {
				invalid := !validUTF8LogRecord(lr)
				if invalid {
					rejected++
				}
				return invalid
			})
			return len(sl.LogRecords) == 0
		})
		return len(rl.ScopeLogs) == 0
	})
	return rejected
}
