// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	math_bits "math/bits"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/plog"
)

// MergeSplit splits and/or merges the provided logs request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *logsRequest) MergeSplit(_ context.Context, cfg exporterbatcher.MaxSizeConfig, r2 Request) ([]Request, error) {
	var req2 *logsRequest
	if r2 != nil {
		var ok bool
		req2, ok = r2.(*logsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
	}

	if cfg.MaxSizeItems == 0 {
		req.setCachedItemsCount(req.ItemsCount() + req2.ItemsCount())
		req2.setCachedItemsCount(0)
		req2.ld.ResourceLogs().MoveAndAppendTo(req.ld.ResourceLogs())
		return []Request{req}, nil
	}

	return req.mergeSplitBasedOnItemCount(cfg, req2)
}

func (req *logsRequest) mergeSplitBasedOnItemCount(cfg exporterbatcher.MaxSizeConfig, req2 *logsRequest) ([]Request, error) {
	var (
		res          []Request
		destReq      *logsRequest
		capacityLeft = cfg.MaxSizeItems
	)
	for _, srcReq := range []*logsRequest{req, req2} {
		if srcReq == nil {
			continue
		}

		srcCount := srcReq.ItemsCount()
		if srcCount <= capacityLeft {
			if destReq == nil {
				destReq = srcReq
			} else {
				destReq.setCachedItemsCount(destReq.ItemsCount() + srcCount)
				srcReq.setCachedItemsCount(0)
				srcReq.ld.ResourceLogs().MoveAndAppendTo(destReq.ld.ResourceLogs())
			}
			capacityLeft -= srcCount
			continue
		}

		for {
			extractedLogs := extractLogsBasedOnItemCount(srcReq.ld, capacityLeft)
			extractedCount := extractedLogs.LogRecordCount()
			if extractedCount == 0 {
				break
			}

			if destReq == nil {
				destReq = &logsRequest{ld: extractedLogs, pusher: srcReq.pusher, cachedItemsCount: extractedCount}
			} else {
				extractedLogs.ResourceLogs().MoveAndAppendTo(destReq.ld.ResourceLogs())
				destReq.setCachedItemsCount(destReq.ItemsCount() + extractedCount)
				srcReq.setCachedItemsCount(srcReq.ItemsCount() - extractedCount)
			}

			// Create new batch once capacity is reached.
			capacityLeft -= extractedCount
			if capacityLeft == 0 {
				res = append(res, destReq)
				destReq = nil
				capacityLeft = cfg.MaxSizeItems
			}
		}
	}

	if destReq != nil {
		res = append(res, destReq)
	}
	return res, nil
}

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogsBasedOnItemCount(srcLogs plog.Logs, count int) plog.Logs {
	destLogs := plog.NewLogs()
	srcLogs.ResourceLogs().RemoveIf(func(srcRL plog.ResourceLogs) bool {
		if count == 0 {
			return false
		}
		needToExtract := resourceLogsCount(srcRL) > count
		if needToExtract {
			srcRL = extractResourceLogsBasedOnItemCount(srcRL, count)
		}
		count -= resourceLogsCount(srcRL)
		srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
		return !needToExtract
	})
	return destLogs
}

// extractResourceLogs extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceLogsBasedOnItemCount(srcRL plog.ResourceLogs, count int) plog.ResourceLogs {
	destRL := plog.NewResourceLogs()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	srcRL.ScopeLogs().RemoveIf(func(srcSL plog.ScopeLogs) bool {
		if count == 0 {
			return false
		}
		needToExtract := srcSL.LogRecords().Len() > count
		if needToExtract {
			srcSL = extractScopeLogsBasedOnItemCount(srcSL, count)
		}
		count -= srcSL.LogRecords().Len()
		srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
		return !needToExtract
	})
	return destRL
}

// extractScopeLogs extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeLogsBasedOnItemCount(srcSL plog.ScopeLogs, count int) plog.ScopeLogs {
	destSL := plog.NewScopeLogs()
	destSL.SetSchemaUrl(srcSL.SchemaUrl())
	srcSL.Scope().CopyTo(destSL.Scope())
	srcSL.LogRecords().RemoveIf(func(srcLR plog.LogRecord) bool {
		if count == 0 {
			return false
		}
		srcLR.MoveTo(destSL.LogRecords().AppendEmpty())
		count--
		return true
	})
	return destSL
}

// resourceLogsCount calculates the total number of log records in the plog.ResourceLogs.
func resourceLogsCount(rl plog.ResourceLogs) int {
	count := 0
	for k := 0; k < rl.ScopeLogs().Len(); k++ {
		count += rl.ScopeLogs().At(k).LogRecords().Len()
	}
	return count
}

func (req *logsRequest) mergeSplitBasedOnByteSize(maxSizeBytes int, req2 *logsRequest) ([]Request, error) {
	var (
		res          []Request
		destReq      *logsRequest
		capacityLeft = maxSizeBytes
	)
	for _, srcReq := range []*logsRequest{req, req2} {
		if srcReq == nil {
			continue
		}

		srcByteSize := srcReq.ByteSize()
		if srcByteSize <= capacityLeft {
			if destReq == nil {
				destReq = srcReq
			} else {
				destReq.setCachedByteSize(destReq.ByteSize() + srcByteSize)
				srcReq.ld.ResourceLogs().MoveAndAppendTo(destReq.ld.ResourceLogs())
			}
			capacityLeft -= srcByteSize
			continue
		}

		for {
			extractedLogs, capacityReached := extractLogsBasedOnByteSize(srcReq.ld, capacityLeft)
			if extractedLogs.LogRecordCount() == 0 {
				break
			}

			extractedByteSize := logsMarshaler.LogsSize(extractedLogs)
			srcReq.setCachedByteSize(srcReq.ByteSize() + extractedByteSize)
			if destReq == nil {
				destReq = &logsRequest{
					ld:             extractedLogs,
					pusher:         srcReq.pusher,
					cachedByteSize: extractedByteSize,
				}
			} else {
				destReq.setCachedByteSize(destReq.ByteSize() + extractedByteSize)
				extractedLogs.ResourceLogs().MoveAndAppendTo(destReq.ld.ResourceLogs())
			}
			// Create new batch once capacity is reached.
			if capacityReached {
				res = append(res, destReq)
				destReq = nil
				capacityLeft = maxSizeBytes
			} else {
				capacityLeft = maxSizeBytes - destReq.ByteSize()
			}
		}
	}

	if destReq != nil {
		res = append(res, destReq)
	}
	return res, nil
}

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogsBasedOnByteSize(srcLogs plog.Logs, capacity int) (plog.Logs, bool) {
	capacityReached := false
	destLogs := plog.NewLogs()
	capacityLeft := capacity - logsMarshaler.LogsSize(destLogs)
	srcLogs.ResourceLogs().RemoveIf(func(srcRL plog.ResourceLogs) bool {
		if capacityReached {
			return false
		}
		needToExtract := logsMarshaler.ResourceLogsSize(srcRL) > capacityLeft
		if needToExtract {
			srcRL, capacityReached = extractResourceLogsBasedOnByteSize(srcRL, capacityLeft)
			if srcRL.ScopeLogs().Len() == 0 {
				return false
			}
		}
		capacityLeft -= deltaCapacity(logsMarshaler.ResourceLogsSize(srcRL))
		srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
		return !needToExtract
	})
	return destLogs, capacityReached
}

// extractResourceLogs extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceLogsBasedOnByteSize(srcRL plog.ResourceLogs, capacity int) (plog.ResourceLogs, bool) {
	capacityReached := false
	destRL := plog.NewResourceLogs()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	capacityLeft := capacity - logsMarshaler.ResourceLogsSize(destRL)
	srcRL.ScopeLogs().RemoveIf(func(srcSL plog.ScopeLogs) bool {
		if capacityReached {
			return false
		}
		needToExtract := logsMarshaler.ScopeLogsSize(srcSL) > capacityLeft
		if needToExtract {
			srcSL, capacityReached = extractScopeLogsBasedOnByteSize(srcSL, capacityLeft)
			if srcSL.LogRecords().Len() == 0 {
				return false
			}
		}

		capacityLeft -= deltaCapacity(logsMarshaler.ScopeLogsSize(srcSL))
		srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
		return !needToExtract
	})
	return destRL, capacityReached
}

// extractScopeLogs extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeLogsBasedOnByteSize(srcSL plog.ScopeLogs, capacity int) (plog.ScopeLogs, bool) {
	capacityReached := false
	destSL := plog.NewScopeLogs()
	destSL.SetSchemaUrl(srcSL.SchemaUrl())
	srcSL.Scope().CopyTo(destSL.Scope())
	capacityLeft := capacity - logsMarshaler.ScopeLogsSize(destSL)

	srcSL.LogRecords().RemoveIf(func(srcLR plog.LogRecord) bool {
		if capacityReached || logsMarshaler.LogRecordSize(srcLR) > capacityLeft {
			capacityReached = true
			return false
		}
		capacityLeft -= deltaCapacity(logsMarshaler.LogRecordSize(srcLR))
		srcLR.MoveTo(destSL.LogRecords().AppendEmpty())
		return true
	})
	return destSL, capacityReached
}

// deltaCapacity() returns the delta size of a proto slice when a new item is added.
// Example:
//
//	proto1 = ProtoMessage()
//	prevSize = proto1.Size()
//	proto1.SomeRepeatedField().AppendEmpty() = proto2
//
// Then, currSize can be calculated as
//
//	currSize = prevSize + deltaCapacity(proto2.Size())
//
// This is derived from gogo/protobuf 's implementation of Size() functions.
func deltaCapacity(newItemSize int) int {
	return 1 + newItemSize + int(math_bits.Len64(uint64(newItemSize|1)+6)/7)
}
