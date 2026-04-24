// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/plog"
)

// MergeSplit splits and/or merges the provided logs request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *logsRequest) MergeSplit(_ context.Context, maxSizePerSizer map[request.SizerType]int64, r2 request.Request) ([]request.Request, error) {
	sizers := make(map[request.SizerType]sizer.LogsSizer)
	for szt := range maxSizePerSizer {
		switch szt {
		case request.SizerTypeItems:
			sizers[szt] = &sizer.LogsCountSizer{}
		case request.SizerTypeBytes:
			sizers[szt] = &sizer.LogsBytesSizer{}
		default:
			return nil, fmt.Errorf("unknown sizer type: %q", szt.String())
		}
	}

	if r2 != nil {
		req2, ok := r2.(*logsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sizers)
	}

	// If no limits we can simply merge the new request into the current and return.
	if len(maxSizePerSizer) == 0 {
		return []request.Request{req}, nil
	}

	return req.split(maxSizePerSizer, sizers)
}

func (req *logsRequest) mergeTo(dst *logsRequest, sizers map[request.SizerType]sizer.LogsSizer) {
	for szt, sz := range sizers {
		dst.setCachedSize(szt, dst.size(szt, sz)+req.size(szt, sz))
		req.setCachedSize(szt, 0)
	}
	req.ld.ResourceLogs().MoveAndAppendTo(dst.ld.ResourceLogs())
}

func (req *logsRequest) split(maxSizePerSizer map[request.SizerType]int64, sizers map[request.SizerType]sizer.LogsSizer) ([]request.Request, error) {
	var res []request.Request
	sortedSzt := getSortedSizerTypes(maxSizePerSizer)
	for req.ld.LogRecordCount() > 0 {
		ld := req.ld
		isInitial := true
		exceededAny := false
		for _, szt := range sortedSzt {
			maxSize := maxSizePerSizer[szt]
			sz := sizers[szt]
			if maxSize > 0 && int64(sz.LogsSize(ld)) > maxSize {
				exceededAny = true
				ldNew := extractLogs(ld, int(maxSize), sz)
				if ldNew.LogRecordCount() == 0 {
					return res, errors.New("one log record size is greater than max size, dropping items")
				}

				if isInitial {
					// ld was req.ld. Remainder is already in req.ld due to in-place modification by extractLogs.
					isInitial = false
				} else {
					// ld was not req.ld. Remainder is in ld, and we must prepend it to req.ld to maintain order.
					newReqLd := plog.NewLogs()
					ld.ResourceLogs().MoveAndAppendTo(newReqLd.ResourceLogs())
					req.ld.ResourceLogs().MoveAndAppendTo(newReqLd.ResourceLogs())
					req.ld = newReqLd
				}

				ld = ldNew
			}
		}

		if !exceededAny {
			break
		}

		req.cachedSizes = make(map[request.SizerType]int)
		res = append(res, newLogsRequest(ld))
	}
	res = append(res, req)
	return res, nil
}

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogs(srcLogs plog.Logs, capacity int, sz sizer.LogsSizer) plog.Logs {
	destLogs := plog.NewLogs()
	capacityLeft := capacity - sz.LogsSize(destLogs)
	srcLogs.ResourceLogs().RemoveIf(func(srcRL plog.ResourceLogs) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRlSize := sz.ResourceLogsSize(srcRL)
		rlSize := sz.DeltaSize(rawRlSize)
		if rlSize > capacityLeft {
			extSrcRL := extractResourceLogs(srcRL, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no log records.
			// Do not add it to the destination if that is the case.
			if extSrcRL.ScopeLogs().Len() > 0 {
				extSrcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
			}
			return extSrcRL.ScopeLogs().Len() != 0
		}
		capacityLeft -= rlSize
		srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
		return true
	})
	return destLogs
}

// extractResourceLogs extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceLogs(srcRL plog.ResourceLogs, capacity int, sz sizer.LogsSizer) plog.ResourceLogs {
	destRL := plog.NewResourceLogs()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceLogsSize(destRL)
	srcRL.ScopeLogs().RemoveIf(func(srcSL plog.ScopeLogs) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawSlSize := sz.ScopeLogsSize(srcSL)
		slSize := sz.DeltaSize(rawSlSize)
		if slSize > capacityLeft {
			extSrcSL := extractScopeLogs(srcSL, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no log records.
			// Do not add it to the destination if that is the case.
			if extSrcSL.LogRecords().Len() > 0 {
				extSrcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
			}
			return extSrcSL.LogRecords().Len() != 0
		}
		capacityLeft -= slSize
		srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
		return true
	})
	return destRL
}

// extractScopeLogs extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeLogs(srcSL plog.ScopeLogs, capacity int, sz sizer.LogsSizer) plog.ScopeLogs {
	destSL := plog.NewScopeLogs()
	destSL.SetSchemaUrl(srcSL.SchemaUrl())
	srcSL.Scope().CopyTo(destSL.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeLogsSize(destSL)
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
		srcLR.MoveTo(destSL.LogRecords().AppendEmpty())
		return true
	})
	return destSL
}

func getSortedSizerTypes[T any](m map[request.SizerType]T) []request.SizerType {
	keys := make([]request.SizerType, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
	return keys
}
