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

package batchprocessor

import (
	"go.opentelemetry.io/collector/model/pdata"
)

// splitLogs removes logrecords from the input data and returns a new data of the specified size.
func splitLogs(size int, src pdata.Logs) pdata.Logs {
	if src.LogRecordCount() <= size {
		return src
	}
	totalCopiedLogs := 0
	dest := pdata.NewLogs()

	src.ResourceLogs().RemoveIf(func(srcRs pdata.ResourceLogs) bool {
		// If we are done skip everything else.
		if totalCopiedLogs == size {
			return false
		}

		destRs := dest.ResourceLogs().AppendEmpty()
		srcRs.Resource().CopyTo(destRs.Resource())

		srcRs.InstrumentationLibraryLogs().RemoveIf(func(srcIlm pdata.InstrumentationLibraryLogs) bool {
			// If we are done skip everything else.
			if totalCopiedLogs == size {
				return false
			}

			destIlm := destRs.InstrumentationLibraryLogs().AppendEmpty()
			srcIlm.InstrumentationLibrary().CopyTo(destIlm.InstrumentationLibrary())

			// If possible to move all metrics do that.
			srcLogsLen := srcIlm.Logs().Len()
			if size >= srcLogsLen+totalCopiedLogs {
				totalCopiedLogs += srcLogsLen
				srcIlm.Logs().MoveAndAppendTo(destIlm.Logs())
				return true
			}

			srcIlm.Logs().RemoveIf(func(srcMetric pdata.LogRecord) bool {
				// If we are done skip everything else.
				if totalCopiedLogs == size {
					return false
				}
				srcMetric.CopyTo(destIlm.Logs().AppendEmpty())
				totalCopiedLogs++
				return true
			})
			return false
		})
		return srcRs.InstrumentationLibraryLogs().Len() == 0
	})

	return dest
}
