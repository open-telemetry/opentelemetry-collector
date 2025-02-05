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
	if r2 != nil {
		req2, ok := r2.(*logsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req)
	}

	// If no limit we can simply merge the new request into the current and return.
	if cfg.MaxSizeItems == 0 && cfg.MaxSizeBytes == 0 {
		return []Request{req}, nil
	}
	return req.split(cfg)
}

func (req *logsRequest) mergeTo(dst *logsRequest) {
	dst.setCachedItemsCount(dst.ItemsCount() + req.ItemsCount())
	req.setCachedItemsCount(0)
	dst.setCachedByteSize(dst.ByteSize() + req.ByteSize())
	req.setCachedByteSize(0)
	req.ld.ResourceLogs().MoveAndAppendTo(dst.ld.ResourceLogs())
}

func (req *logsRequest) split(cfg exporterbatcher.MaxSizeConfig) ([]Request, error) {
	var res []Request
	if cfg.MaxSizeItems != 0 {
		for req.ItemsCount() > cfg.MaxSizeItems {
			ld := extractLogs(req.ld, cfg.MaxSizeItems)
			size := ld.LogRecordCount()
			req.setCachedItemsCount(req.ItemsCount() - size)
			res = append(res, &logsRequest{ld: ld, pusher: req.pusher, cachedItemsCount: size})
		}
	} else if cfg.MaxSizeBytes != 0 {
		for req.ByteSize() > cfg.MaxSizeBytes {
			ld := extractLogsBasedOnByteSize(req.ld, cfg.MaxSizeBytes)
			size := logsMarshaler.LogsSize(ld)
			req.setCachedByteSize(req.ByteSize() - size)
			res = append(res, &logsRequest{ld: ld, pusher: req.pusher, cachedByteSize: size})
		}
	}
	res = append(res, req)
	return res, nil
}

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogs(srcLogs plog.Logs, count int) plog.Logs {
	destLogs := plog.NewLogs()
	srcLogs.ResourceLogs().RemoveIf(func(srcRL plog.ResourceLogs) bool {
		if count == 0 {
			return false
		}
		needToExtract := resourceLogsCount(srcRL) > count
		if needToExtract {
			srcRL = extractResourceLogs(srcRL, count)
		}
		count -= resourceLogsCount(srcRL)
		srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
		return !needToExtract
	})
	return destLogs
}

// extractResourceLogs extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceLogs(srcRL plog.ResourceLogs, count int) plog.ResourceLogs {
	destRL := plog.NewResourceLogs()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	srcRL.ScopeLogs().RemoveIf(func(srcSL plog.ScopeLogs) bool {
		if count == 0 {
			return false
		}
		needToExtract := srcSL.LogRecords().Len() > count
		if needToExtract {
			srcSL = extractScopeLogs(srcSL, count)
		}
		count -= srcSL.LogRecords().Len()
		srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
		return !needToExtract
	})
	return destRL
}

// extractScopeLogs extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeLogs(srcSL plog.ScopeLogs, count int) plog.ScopeLogs {
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

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogsBasedOnByteSize(srcLogs plog.Logs, capacity int) plog.Logs {
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
	return destLogs
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
//	prevSize := proto1.Size()
//	proto1.RepeatedField().AppendEmpty() = proto2
//
// Then currSize of proto1 can be calculated as
//
//	currSize := (prevSize + deltaCapacity(proto2.Size()))
//
// This is derived from gogo/protobuf 's Size() function
func deltaCapacity(newItemSize int) int {
	return 1 + newItemSize + int(math_bits.Len64(uint64(newItemSize|1)+6)/7)
}
