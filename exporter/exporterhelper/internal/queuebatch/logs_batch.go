// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"
	"math"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/plog"
)

// MergeSplit splits and/or merges the provided logs request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *logsRequest) MergeSplit(_ context.Context, limits map[request.SizerType]int64, r2 request.Request) ([]request.Request, error) {
	sizers := make(map[request.SizerType]sizer.LogsSizer)
	for szt := range limits {
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
	if len(limits) == 0 {
		return []request.Request{req}, nil
	}

	return req.split(limits, sizers)
}

func (req *logsRequest) mergeTo(dst *logsRequest, sizers map[request.SizerType]sizer.LogsSizer) {
	for szt, sz := range sizers {
		dst.setCachedSize(szt, dst.size(szt, sz)+req.size(szt, sz))
		req.setCachedSize(szt, 0)
	}
	req.ld.ResourceLogs().MoveAndAppendTo(dst.ld.ResourceLogs())
}

func (req *logsRequest) split(limits map[request.SizerType]int64, sizers map[request.SizerType]sizer.LogsSizer) ([]request.Request, error) {
	var res []request.Request
	for {
		exceeded := false
		for szt, limit := range limits {
			sz := sizers[szt]
			if limit > 0 && int64(req.size(szt, sz)) > limit {
				exceeded = true
				break
			}
		}
		if !exceeded {
			break
		}

		intLimits := make(map[request.SizerType]int)
		for szt, limit := range limits {
			intLimits[szt] = int(limit)
		}

		ld, _ := extractLogs(req.ld, intLimits, sizers)
		if ld.LogRecordCount() == 0 {
			return res, fmt.Errorf("one log record size is greater than max size, dropping items")
		}
		req.cachedSizes = nil
		res = append(res, newLogsRequest(ld))
	}
	res = append(res, req)
	return res, nil
}

// extractLogs extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractLogs(srcLogs plog.Logs, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.LogsSizer) (plog.Logs, map[request.SizerType]int) {
	destLogs := plog.NewLogs()
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)

	for szt, limit := range limits {
		sz := sizers[szt]
		if limit == 0 {
			capacityLeft[szt] = math.MaxInt
		} else {
			capacityLeft[szt] = limit - sz.LogsSize(destLogs)
		}
		removedSizes[szt] = 0
	}

	srcLogs.ResourceLogs().RemoveIf(func(srcRL plog.ResourceLogs) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rawRlSize := sz.ResourceLogsSize(srcRL)
			rlSize := sz.DeltaSize(rawRlSize)
			if rlSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			extSrcRL, extRlSizes := extractResourceLogs(srcRL, capacityLeft, sizers)
			
			for szt, sz := range sizers {
				extRlSize := extRlSizes[szt]
				capacityLeft[szt] = 0
				removedSizes[szt] += extRlSize
				
				rawRlSize := sz.ResourceLogsSize(srcRL)
				rlSize := sz.DeltaSize(rawRlSize)
				removedSizes[szt] += rlSize - rawRlSize - (sz.DeltaSize(rawRlSize-extRlSize) - (rawRlSize - extRlSize))
			}
			
			if extSrcRL.ScopeLogs().Len() > 0 {
				extSrcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
			}
			return extSrcRL.ScopeLogs().Len() != 0
		}

		for szt, sz := range sizers {
			rawRlSize := sz.ResourceLogsSize(srcRL)
			rlSize := sz.DeltaSize(rawRlSize)
			capacityLeft[szt] -= rlSize
			removedSizes[szt] += rlSize
		}
		srcRL.MoveTo(destLogs.ResourceLogs().AppendEmpty())
		return true
	})
	return destLogs, removedSizes
}

// extractResourceLogs extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceLogs(srcRL plog.ResourceLogs, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.LogsSizer) (plog.ResourceLogs, map[request.SizerType]int) {
	destRL := plog.NewResourceLogs()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)
	
	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.ResourceLogsSize(destRL)
		removedSizes[szt] = 0
	}

	srcRL.ScopeLogs().RemoveIf(func(srcSL plog.ScopeLogs) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rawSlSize := sz.ScopeLogsSize(srcSL)
			slSize := sz.DeltaSize(rawSlSize)
			if slSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			extSrcSL, extSlSizes := extractScopeLogs(srcSL, capacityLeft, sizers)
			
			for szt, sz := range sizers {
				extSlSize := extSlSizes[szt]
				capacityLeft[szt] = 0
				removedSizes[szt] += extSlSize
				
				rawSlSize := sz.ScopeLogsSize(srcSL)
				slSize := sz.DeltaSize(rawSlSize)
				removedSizes[szt] += slSize - rawSlSize - (sz.DeltaSize(rawSlSize-extSlSize) - (rawSlSize - extSlSize))
			}
			
			if extSrcSL.LogRecords().Len() > 0 {
				extSrcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
			}
			return extSrcSL.LogRecords().Len() != 0
		}

		for szt, sz := range sizers {
			rawSlSize := sz.ScopeLogsSize(srcSL)
			slSize := sz.DeltaSize(rawSlSize)
			capacityLeft[szt] -= slSize
			removedSizes[szt] += slSize
		}
		srcSL.MoveTo(destRL.ScopeLogs().AppendEmpty())
		return true
	})
	return destRL, removedSizes
}

// extractScopeLogs extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeLogs(srcSL plog.ScopeLogs, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.LogsSizer) (plog.ScopeLogs, map[request.SizerType]int) {
	destSL := plog.NewScopeLogs()
	destSL.SetSchemaUrl(srcSL.SchemaUrl())
	srcSL.Scope().CopyTo(destSL.Scope())
	
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)
	
	for szt, limit := range limits {
		sz := sizers[szt]
		if limit == 0 {
			capacityLeft[szt] = math.MaxInt
		} else {
			capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.ScopeLogsSize(destSL)
		}
		removedSizes[szt] = 0
	}

	srcSL.LogRecords().RemoveIf(func(srcLR plog.LogRecord) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rlSize := sz.DeltaSize(sz.LogRecordSize(srcLR))
			if rlSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			return false
		}

		for szt, sz := range sizers {
			rlSize := sz.DeltaSize(sz.LogRecordSize(srcLR))
			capacityLeft[szt] -= rlSize
			removedSizes[szt] += rlSize
		}
		srcLR.MoveTo(destSL.LogRecords().AppendEmpty())
		return true
	})
	return destSL, removedSizes
}
