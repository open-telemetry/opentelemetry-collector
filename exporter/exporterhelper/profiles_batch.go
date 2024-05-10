// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// mergeProfiles merges two logs requests into one.
func mergeProfiles(_ context.Context, r1 Request, r2 Request) (Request, error) {
	lr1, ok1 := r1.(*profilesRequest)
	lr2, ok2 := r2.(*profilesRequest)
	if !ok1 || !ok2 {
		return nil, errors.New("invalid input type")
	}
	lr2.pd.ResourceProfiles().MoveAndAppendTo(lr1.pd.ResourceProfiles())
	return lr1, nil
}

// mergeSplitProfiles splits and/or merges the logs into multiple requests based on the MaxSizeConfig.
func mergeSplitProfiles(_ context.Context, cfg exporterbatcher.MaxSizeConfig, r1 Request, r2 Request) ([]Request, error) {
	var (
		res          []Request
		destReq      *profilesRequest
		capacityLeft = cfg.MaxSizeItems
	)
	for _, req := range []Request{r1, r2} {
		if req == nil {
			continue
		}
		srcReq, ok := req.(*profilesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		if srcReq.pd.ProfileCount() <= capacityLeft {
			if destReq == nil {
				destReq = srcReq
			} else {
				srcReq.pd.ResourceProfiles().MoveAndAppendTo(destReq.pd.ResourceProfiles())
			}
			capacityLeft -= destReq.pd.ProfileCount()
			continue
		}

		for {
			extractedProfiles := extractProfiles(srcReq.pd, capacityLeft)
			if extractedProfiles.ProfileCount() == 0 {
				break
			}
			capacityLeft -= extractedProfiles.ProfileCount()
			if destReq == nil {
				destReq = &profilesRequest{pd: extractedProfiles, pusher: srcReq.pusher}
			} else {
				extractedProfiles.ResourceProfiles().MoveAndAppendTo(destReq.pd.ResourceProfiles())
			}
			// Create new batch once capacity is reached.
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

// extractProfiles extracts logs from the input logs and returns a new logs with the specified number of log records.
func extractProfiles(srcProfiles pprofile.Profiles, count int) pprofile.Profiles {
	destProfiles := pprofile.NewProfiles()
	srcProfiles.ResourceProfiles().RemoveIf(func(srcRL pprofile.ResourceProfiles) bool {
		if count == 0 {
			return false
		}
		needToExtract := resourceProfilesCount(srcRL) > count
		if needToExtract {
			srcRL = extractResourceProfiles(srcRL, count)
		}
		count -= resourceProfilesCount(srcRL)
		srcRL.MoveTo(destProfiles.ResourceProfiles().AppendEmpty())
		return !needToExtract
	})
	return destProfiles
}

// extractResourceProfiles extracts resource logs and returns a new resource logs with the specified number of log records.
func extractResourceProfiles(srcRL pprofile.ResourceProfiles, count int) pprofile.ResourceProfiles {
	destRL := pprofile.NewResourceProfiles()
	destRL.SetSchemaUrl(srcRL.SchemaUrl())
	srcRL.Resource().CopyTo(destRL.Resource())
	srcRL.ScopeProfiles().RemoveIf(func(srcSL pprofile.ScopeProfiles) bool {
		if count == 0 {
			return false
		}
		needToExtract := srcSL.Profiles().Len() > count
		if needToExtract {
			srcSL = extractScopeProfiles(srcSL, count)
		}
		count -= srcSL.Profiles().Len()
		srcSL.MoveTo(destRL.ScopeProfiles().AppendEmpty())
		return !needToExtract
	})
	return destRL
}

// extractScopeProfiles extracts scope logs and returns a new scope logs with the specified number of log records.
func extractScopeProfiles(srcSL pprofile.ScopeProfiles, count int) pprofile.ScopeProfiles {
	destSL := pprofile.NewScopeProfiles()
	destSL.SetSchemaUrl(srcSL.SchemaUrl())
	srcSL.Scope().CopyTo(destSL.Scope())
	srcSL.Profiles().RemoveIf(func(srcLR pprofile.ProfileContainer) bool {
		if count == 0 {
			return false
		}
		srcLR.MoveTo(destSL.Profiles().AppendEmpty())
		count--
		return true
	})
	return destSL
}

// resourceProfilesCount calculates the total number of log records in the pprofile.ResourceProfiles.
func resourceProfilesCount(rl pprofile.ResourceProfiles) int {
	count := 0
	for k := 0; k < rl.ScopeProfiles().Len(); k++ {
		count += rl.ScopeProfiles().At(k).Profiles().Len()
	}
	return count
}
