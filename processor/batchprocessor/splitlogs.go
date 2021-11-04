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

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/collector/model/pdata"
)

// splitLogs removes logrecords from the input data and returns a new data of the specified size.
func splitLogs(size int, src pdata.Logs) pdata.Logs {
	if src.LogRecordCount() <= size {
		return src
	}
	totalCopiedLogRecords := 0
	dest := pdata.NewLogs()

	src.ResourceLogs().RemoveIf(func(srcRl pdata.ResourceLogs) bool {
		// If we are done skip everything else.
		if totalCopiedLogRecords == size {
			return false
		}

		// If it fully fits
		srcRlLRC := resourceLRC(srcRl)
		if (totalCopiedLogRecords + srcRlLRC) <= size {
			totalCopiedLogRecords += srcRlLRC
			srcRl.MoveTo(dest.ResourceLogs().AppendEmpty())
			return true
		}

		destRl := dest.ResourceLogs().AppendEmpty()
		srcRl.Resource().CopyTo(destRl.Resource())
		srcRl.InstrumentationLibraryLogs().RemoveIf(func(srcIll pdata.InstrumentationLibraryLogs) bool {
			// If we are done skip everything else.
			if totalCopiedLogRecords == size {
				return false
			}

			// If possible to move all metrics do that.
			srcIllLRC := srcIll.Logs().Len()
			if size >= srcIllLRC+totalCopiedLogRecords {
				totalCopiedLogRecords += srcIllLRC
				srcIll.MoveTo(destRl.InstrumentationLibraryLogs().AppendEmpty())
				return true
			}

			destIll := destRl.InstrumentationLibraryLogs().AppendEmpty()
			srcIll.InstrumentationLibrary().CopyTo(destIll.InstrumentationLibrary())
			srcIll.Logs().RemoveIf(func(srcMetric pdata.LogRecord) bool {
				// If we are done skip everything else.
				if totalCopiedLogRecords == size {
					return false
				}
				srcMetric.MoveTo(destIll.Logs().AppendEmpty())
				totalCopiedLogRecords++
				return true
			})
			return false
		})
		return srcRl.InstrumentationLibraryLogs().Len() == 0
	})

	return dest
}

// resourceLRC calculates the total number of log records in the pdata.ResourceLogs.
func resourceLRC(rs pdata.ResourceLogs) (count int) {
	for k := 0; k < rs.InstrumentationLibraryLogs().Len(); k++ {
		count += rs.InstrumentationLibraryLogs().At(k).Logs().Len()
	}
	return
}
