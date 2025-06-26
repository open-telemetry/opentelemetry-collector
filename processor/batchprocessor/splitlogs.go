// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/collector/pdata/plog"
)

// splitLogs removes logrecords from the input data and returns a new data of the specified size.
func splitLogs(size int, src plog.Logs) plog.Logs {
	if !hasAtLeastNRecords(src, size+1) {
		return src
	}
	totalCopiedLogRecords := 0
	dest := plog.NewLogs()

	src.ResourceLogs().RemoveIf(func(srcRl plog.ResourceLogs) bool {
		// If we are done skip everything else.
		if totalCopiedLogRecords == size {
			return false
		}

		// If it fully fits
		wontFit, srcRlLRC := resourceHasAtLeastNRecords(srcRl, size-totalCopiedLogRecords+1)
		if !wontFit {
			totalCopiedLogRecords += srcRlLRC
			srcRl.MoveTo(dest.ResourceLogs().AppendEmpty())
			return true
		}

		destRl := dest.ResourceLogs().AppendEmpty()
		srcRl.Resource().CopyTo(destRl.Resource())
		srcRl.ScopeLogs().RemoveIf(func(srcIll plog.ScopeLogs) bool {
			// If we are done skip everything else.
			if totalCopiedLogRecords == size {
				return false
			}

			// If possible to move all metrics do that.
			srcIllLRC := srcIll.LogRecords().Len()
			if size >= srcIllLRC+totalCopiedLogRecords {
				totalCopiedLogRecords += srcIllLRC
				srcIll.MoveTo(destRl.ScopeLogs().AppendEmpty())
				return true
			}

			destIll := destRl.ScopeLogs().AppendEmpty()
			srcIll.Scope().CopyTo(destIll.Scope())
			srcIll.LogRecords().RemoveIf(func(srcMetric plog.LogRecord) bool {
				// If we are done skip everything else.
				if totalCopiedLogRecords == size {
					return false
				}
				srcMetric.MoveTo(destIll.LogRecords().AppendEmpty())
				totalCopiedLogRecords++
				return true
			})
			return false
		})
		return srcRl.ScopeLogs().Len() == 0
	})

	return dest
}

func resourceHasAtLeastNRecords(rs plog.ResourceLogs, n int) (bool, int) {
	count := 0
	for k := 0; k < rs.ScopeLogs().Len(); k++ {
		count += rs.ScopeLogs().At(k).LogRecords().Len()
		if count >= n {
			return true, 0
		}
	}
	return false, count
}

func hasAtLeastNRecords(ms plog.Logs, n int) bool {
	logCount := 0
	rss := ms.ResourceLogs()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ill := rs.ScopeLogs()
		for i := 0; i < ill.Len(); i++ {
			logs := ill.At(i)
			logCount += logs.LogRecords().Len()
			if logCount >= n {
				return true
			}
		}
	}
	return logCount >= n
}
