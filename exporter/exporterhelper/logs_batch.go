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

func sizerFromConfig(cfg exporterbatcher.SizeConfig) sizer.LogsSizer {
	switch cfg.Sizer {
	case exporterbatcher.SizerTypeItems:
		return &sizer.LogsCountSizer{}
	case exporterbatcher.SizerTypeBytes:
		return &sizer.LogsBytesSizer{}
	default:
		return &sizer.LogsCountSizer{}
	}
}

// MergeSplit splits and/or merges the provided logs request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *logsRequest) MergeSplit(_ context.Context, cfg exporterbatcher.SizeConfig, r2 Request) ([]Request, error) {
	sizer := sizerFromConfig(cfg)
	if r2 != nil {
		req2, ok := r2.(*logsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sizer)
	}

	// If no limit we can simply merge the new request into the current and return.
	if cfg.MaxSize == 0 {
		return []Request{req}, nil
	}

	return req.split(cfg.MaxSize, sizer), nil
}

func (req *logsRequest) mergeTo(dst *logsRequest, sizer sizer.LogsSizer) {
	if sizer != nil {
		dst.setCachedSize(dst.Size(sizer) + req.Size(sizer))
		req.setCachedSize(0)
	}
	req.ld.ResourceLogs().MoveAndAppendTo(dst.ld.ResourceLogs())
}

func (req *logsRequest) split(maxSize int, sizer sizer.LogsSizer) []Request {
	var res []Request
	for req.Size(sizer) > maxSize {
		ld := extractLogs(req.ld, maxSize, sizer)
		size := sizer.LogsSize(ld)
		req.setCachedSize(req.Size(sizer) - size)
		res = append(res, &logsRequest{ld: ld, pusher: req.pusher, cachedSize: size})
	}
	res = append(res, req)
	return res
}

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogs(srcLogs plog.Logs, capacity int, sizer sizer.LogsSizer) plog.Logs {
	capacityReached := false
	destLogs := plog.NewLogs()
	capacityLeft := capacity - sizer.LogsSize(destLogs)
	srcLogs.ResourceLogs().RemoveIf(func(srcRL plog.ResourceLogs) bool {
		if capacityReached {
			return false
		}
		needToExtract := sizer.ResourceLogsSize(srcRL) > capacityLeft
		if needToExtract {
			srcRL, capacityReached = extractResourceLogs(srcRL, capacityLeft, sizer)
			if srcRL.ScopeLogs().Len() == 0 {
				return false
			}
		}
		capacityLeft -= sizer.DeltaSize(sizer.ResourceLogsSize(srcRL))
		srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
		return !needToExtract
	})
	return destLogs
}

// extractResourceLogs extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceLogs(srcRL plog.ResourceLogs, capacity int, sizer sizer.LogsSizer) (plog.ResourceLogs, bool) {
	capacityReached := false
	destRL := plog.NewResourceLogs()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	capacityLeft := capacity - sizer.ResourceLogsSize(destRL)
	srcRL.ScopeLogs().RemoveIf(func(srcSL plog.ScopeLogs) bool {
		if capacityReached {
			return false
		}
		needToExtract := sizer.ScopeLogsSize(srcSL) > capacityLeft
		if needToExtract {
			srcSL, capacityReached = extractScopeLogs(srcSL, capacityLeft, sizer)
			if srcSL.LogRecords().Len() == 0 {
				return false
			}
		}
		capacityLeft -= sizer.DeltaSize(sizer.ScopeLogsSize(srcSL))
		srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
		return !needToExtract
	})
	return destRL, capacityReached
}

// extractScopeLogs extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeLogs(srcSL plog.ScopeLogs, capacity int, sizer sizer.LogsSizer) (plog.ScopeLogs, bool) {
	capacityReached := false
	destSL := plog.NewScopeLogs()
	destSL.SetSchemaUrl(srcSL.SchemaUrl())
	srcSL.Scope().CopyTo(destSL.Scope())
	capacityLeft := capacity - sizer.ScopeLogsSize(destSL)
	srcSL.LogRecords().RemoveIf(func(srcLR plog.LogRecord) bool {
		if capacityReached || sizer.LogRecordSize(srcLR) > capacityLeft {
			capacityReached = true
			return false
		}
		capacityLeft -= sizer.DeltaSize(sizer.LogRecordSize(srcLR))
		srcLR.MoveTo(destSL.LogRecords().AppendEmpty())
		return true
	})
	return destSL, capacityReached
}
