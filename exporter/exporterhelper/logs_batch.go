// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/plog"
)

// MergeSplit splits and/or merges the provided logs request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *logsRequest) MergeSplit(_ context.Context, cfg exporterbatcher.SizeConfig, r2 Request) ([]Request, error) {
	var sz sizer.LogsSizer
	switch cfg.Sizer {
	case exporterbatcher.SizerTypeItems:
		sz = &sizer.LogsCountSizer{}
	case exporterbatcher.SizerTypeBytes:
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
	if cfg.MaxSize == 0 {
		return []Request{req}, nil
	}

	return req.split(cfg.MaxSize, sz), nil
}

func (req *logsRequest) mergeTo(dst *logsRequest, sz sizer.LogsSizer) {
	if sz != nil {
		dst.setCachedSize(dst.Size(sz) + req.Size(sz))
		req.setCachedSize(0)
	}
	req.ld.ResourceLogs().MoveAndAppendTo(dst.ld.ResourceLogs())
}

func (req *logsRequest) split(maxSize int, sz sizer.LogsSizer) []Request {
	var res []Request
	for req.Size(sz) > maxSize {
		ld := extractLogs(req.ld, maxSize, sz)
		size := sz.LogsSize(ld)
		req.setCachedSize(req.Size(sz) - size)
		res = append(res, &logsRequest{ld: ld, pusher: req.pusher, cachedSize: size})
	}
	res = append(res, req)
	return res
}

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogs(srcLogs plog.Logs, capacity int, sizer sizer.LogsSizer) plog.Logs {
	destLogs := plog.NewLogs()
	capacityLeft := capacity - sizer.LogsSize(destLogs)
	srcLogs.ResourceLogs().RemoveIf(func(srcRL plog.ResourceLogs) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rlSize := sizer.DeltaSize(sizer.ResourceLogsSize(srcRL))
		if rlSize > capacityLeft {
			srcRL = extractResourceLogs(srcRL, capacityLeft, sizer)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no log records.
			// Do not add it to the destination if that is the case.
			if srcRL.ScopeLogs().Len() > 0 {
				srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
			}
			return srcRL.ScopeLogs().Len() != 0
		}
		capacityLeft -= rlSize
		srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
		return true
	})
	return destLogs
}

// extractResourceLogs extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceLogs(srcRL plog.ResourceLogs, capacity int, sizer sizer.LogsSizer) plog.ResourceLogs {
	destRL := plog.NewResourceLogs()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	capacityLeft := capacity - sizer.ResourceLogsSize(destRL)
	srcRL.ScopeLogs().RemoveIf(func(srcSL plog.ScopeLogs) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		slSize := sizer.DeltaSize(sizer.ScopeLogsSize(srcSL))
		if slSize > capacityLeft {
			srcSL = extractScopeLogs(srcSL, capacityLeft, sizer)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no log records.
			// Do not add it to the destination if that is the case.
			if srcSL.LogRecords().Len() > 0 {
				srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
			}
			return srcSL.LogRecords().Len() != 0
		}
		capacityLeft -= slSize
		srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
		return true
	})
	return destRL
}

// extractScopeLogs extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeLogs(srcSL plog.ScopeLogs, capacity int, sizer sizer.LogsSizer) plog.ScopeLogs {
	destSL := plog.NewScopeLogs()
	destSL.SetSchemaUrl(srcSL.SchemaUrl())
	srcSL.Scope().CopyTo(destSL.Scope())
	capacityLeft := capacity - sizer.ScopeLogsSize(destSL)
	srcSL.LogRecords().RemoveIf(func(srcLR plog.LogRecord) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rlSize := sizer.DeltaSize(sizer.LogRecordSize(srcLR))
		if rlSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rlSize
		srcLR.MoveTo(destSL.LogRecords().AppendEmpty())
		return true
	})
	return destSL
}
