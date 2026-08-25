// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/plog"
)

// MergeSplit splits and/or merges the provided logs request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *logsRequest) MergeSplit(_ context.Context, maxSize int, szt request.SizerType, r2 request.Request) ([]request.Request, error) {
	var sz sizer.LogsSizer
	switch szt {
	case request.SizerTypeItems:
		sz = &sizer.LogsCountSizer{}
	case request.SizerTypeBytes:
		sz = &sizer.LogsBytesSizer{}
	default:
		return nil, errors.New("unknown sizer type")
	}
	if r2 != nil {
		req2, ok := r2.(*logsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sz)
	}

	// If no limit we can simply merge the new request into the current and return.
	if maxSize == 0 {
		return []request.Request{req}, nil
	}

	return req.split(maxSize, sz)
}

func (req *logsRequest) mergeTo(dst *logsRequest, sz sizer.LogsSizer) {
	if sz != nil {
		dst.setCachedSize(dst.size(sz) + req.size(sz))
		req.setCachedSize(0)
	}
	req.ld.ResourceLogs().MoveAndAppendTo(dst.ld.ResourceLogs())
}

func (req *logsRequest) split(maxSize int, sz sizer.LogsSizer) ([]request.Request, error) {
	var res []request.Request
	for req.size(sz) > maxSize {
		ld, removedSize := extractLogs(req.ld, maxSize, sz)
		if ld.LogRecordCount() == 0 {
			return res, fmt.Errorf("one log record size is greater than max size, dropping items: %d", req.ld.LogRecordCount())
		}
		req.setCachedSize(req.size(sz) - removedSize)
		res = append(res, newLogsRequest(ld))
	}
	res = append(res, req)
	return res, nil
}

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogs(srcLogs plog.Logs, capacity int, sz sizer.LogsSizer) (plog.Logs, int) {
	destLogs := plog.NewLogs()
	capacityLeft := capacity - sz.LogsSize(destLogs)
	removedSize := 0
	srcLogs.ResourceLogs().RemoveIf(func(srcRL plog.ResourceLogs) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRlSize := sz.ResourceLogsSize(srcRL)
		rlSize := sz.DeltaSize(rawRlSize)
		if rlSize > capacityLeft {
			extSrcRL, extRlSize := extractResourceLogs(srcRL, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extRlSize
			// There represents the delta between the delta sizes.
			removedSize += rlSize - rawRlSize - (sz.DeltaSize(rawRlSize-extRlSize) - (rawRlSize - extRlSize))
			// It is possible that for the bytes scenario, the extracted field contains no log records.
			// Do not add it to the destination if that is the case.
			if extSrcRL.ScopeLogs().Len() > 0 {
				extSrcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
			}
			return extSrcRL.ScopeLogs().Len() != 0
		}
		capacityLeft -= rlSize
		removedSize += rlSize
		srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
		return true
	})
	return destLogs, removedSize
}

// extractResourceLogs extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceLogs(srcRL plog.ResourceLogs, capacity int, sz sizer.LogsSizer) (plog.ResourceLogs, int) {
	destRL := plog.NewResourceLogs()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceLogsSize(destRL)
	removedSize := 0
	srcRL.ScopeLogs().RemoveIf(func(srcSL plog.ScopeLogs) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawSlSize := sz.ScopeLogsSize(srcSL)
		slSize := sz.DeltaSize(rawSlSize)
		if slSize > capacityLeft {
			extSrcSL, extSlSize := extractScopeLogs(srcSL, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extSlSize
			// There represents the delta between the delta sizes.
			removedSize += slSize - rawSlSize - (sz.DeltaSize(rawSlSize-extSlSize) - (rawSlSize - extSlSize))
			// It is possible that for the bytes scenario, the extracted field contains no log records.
			// Do not add it to the destination if that is the case.
			if extSrcSL.LogRecords().Len() > 0 {
				extSrcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
			}
			return extSrcSL.LogRecords().Len() != 0
		}
		capacityLeft -= slSize
		removedSize += slSize
		srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
		return true
	})
	return destRL, removedSize
}

// extractScopeLogs extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeLogs(srcSL plog.ScopeLogs, capacity int, sz sizer.LogsSizer) (plog.ScopeLogs, int) {
	destSL := plog.NewScopeLogs()
	destSL.SetSchemaUrl(srcSL.SchemaUrl())
	srcSL.Scope().CopyTo(destSL.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeLogsSize(destSL)
	removedSize := 0
	srcSL.LogRecords().RemoveIf(func(srcLR plog.LogRecord) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rlSize := sz.DeltaSize(sz.LogRecordSize(srcLR))
		if rlSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rlSize
		removedSize += rlSize
		srcLR.MoveTo(destSL.LogRecords().AppendEmpty())
		return true
	})
	return destSL, removedSize
}
