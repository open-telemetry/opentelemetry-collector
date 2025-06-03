// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// MergeSplit splits and/or merges the profiles into multiple requests based on the MaxSizeConfig.
//
// Following the OTLP 1.7.0 upgrade, this is currently a noop.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/13106
func (req *profilesRequest) MergeSplit(_ context.Context, maxSize int, szt exporterhelper.RequestSizerType, r2 exporterhelper.Request) ([]exporterhelper.Request, error) {
	var sz sizer.ProfilesSizer
	switch szt {
	case exporterhelper.RequestSizerTypeItems:
		sz = &sizer.ProfilesCountSizer{}
	case exporterhelper.RequestSizerTypeBytes:
		sz = &sizer.ProfilesBytesSizer{}
	default:
		return nil, errors.New("unknown sizer type")
	}

	if r2 != nil && r2.ItemsCount() > 0 {
		req2, ok := r2.(*profilesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		// TODO(13106): handle merging of profiles (and change the indice tables with their new indices)
		// req2.mergeTo(req, sz)

		// If no limit we can simply merge the new request into the current and return.
		if maxSize == 0 {
			return []exporterhelper.Request{req, req2}, nil
		}

		return append(req.split(maxSize, sz), req2.split(maxSize, sz)...), nil
	}

	// If no limit we can simply merge the new request into the current and return.
	if maxSize == 0 {
		return []exporterhelper.Request{req}, nil
	}
	return req.split(maxSize, sz), nil
}

// TODO(13106): handle merging of profiles (and change the indice tables with their new indices)
/*func (req *profilesRequest) mergeTo(dst *profilesRequest, sz sizer.ProfilesSizer) {
	if sz != nil {
		dst.setCachedSize(dst.size(sz) + req.size(sz))
		req.setCachedSize(0)
	}
	req.pd.ResourceProfiles().MoveAndAppendTo(dst.pd.ResourceProfiles())
}*/

func (req *profilesRequest) split(maxSize int, sz sizer.ProfilesSizer) []exporterhelper.Request {
	var res []exporterhelper.Request
	for req.size(sz) > maxSize {
		pd, rmSize := extractProfiles(req.pd, maxSize, sz)
		req.setCachedSize(req.size(sz) - rmSize)
		res = append(res, newProfilesRequest(pd))
	}
	res = append(res, req)
	return res
}

// extractProfiles extracts a new profiles with a maximum number of samples.
func extractProfiles(srcProfiles pprofile.Profiles, capacity int, sz sizer.ProfilesSizer) (pprofile.Profiles, int) {
	destProfiles := pprofile.NewProfiles()
	capacityLeft := capacity - sz.ProfilesSize(destProfiles)
	removedSize := 0
	srcProfiles.ResourceProfiles().RemoveIf(func(srcRP pprofile.ResourceProfiles) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRpSize := sz.ResourceProfilesSize(srcRP)
		rpSize := sz.DeltaSize(rawRpSize)

		if rpSize > capacityLeft {
			extSrcRP, extRpSize := extractResourceProfiles(srcRP, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extRpSize
			// There represents the delta between the delta sizes.
			removedSize += rpSize - rawRpSize - (sz.DeltaSize(rawRpSize-extRpSize) - (rawRpSize - extRpSize))
			// It is possible that for the bytes scenario, the extracted field contains no profiles.
			// Do not add it to the destination if that is the case.
			if extSrcRP.ScopeProfiles().Len() > 0 {
				extSrcRP.MoveTo(destProfiles.ResourceProfiles().AppendEmpty())
			}
			return extSrcRP.ScopeProfiles().Len() != 0
		}
		capacityLeft -= rpSize
		removedSize += rpSize
		srcRP.MoveTo(destProfiles.ResourceProfiles().AppendEmpty())
		return true
	})
	return destProfiles, removedSize
}

// extractResourceProfiles extracts profiles and returns a new resource profiles with the specified number of profiles.
func extractResourceProfiles(srcRP pprofile.ResourceProfiles, capacity int, sz sizer.ProfilesSizer) (pprofile.ResourceProfiles, int) {
	destRP := pprofile.NewResourceProfiles()
	destRP.SetSchemaUrl(srcRP.SchemaUrl())
	srcRP.Resource().CopyTo(destRP.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceProfilesSize(destRP)
	removedSize := 0

	srcRP.ScopeProfiles().RemoveIf(func(srcSS pprofile.ScopeProfiles) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rawSlSize := sz.ScopeProfilesSize(srcSS)
		ssSize := sz.DeltaSize(rawSlSize)
		if ssSize > capacityLeft {
			extSrcSS, extSsSize := extractScopeProfiles(srcSS, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extSsSize
			// There represents the delta between the delta sizes.
			removedSize += ssSize - rawSlSize - (sz.DeltaSize(rawSlSize-extSsSize) - (rawSlSize - extSsSize))
			// It is possible that for the bytes scenario, the extracted field contains no profiles.
			// Do not add it to the destination if that is the case.
			if extSrcSS.Profiles().Len() > 0 {
				extSrcSS.MoveTo(destRP.ScopeProfiles().AppendEmpty())
			}
			return extSrcSS.Profiles().Len() != 0
		}
		capacityLeft -= ssSize
		removedSize += ssSize
		srcSS.MoveTo(destRP.ScopeProfiles().AppendEmpty())
		return true
	})

	return destRP, removedSize
}

// extractScopeProfiles extracts profiles and returns a new scope profiles with the specified number of profiles.
func extractScopeProfiles(srcSS pprofile.ScopeProfiles, capacity int, sz sizer.ProfilesSizer) (pprofile.ScopeProfiles, int) {
	destSS := pprofile.NewScopeProfiles()
	destSS.SetSchemaUrl(srcSS.SchemaUrl())
	srcSS.Scope().CopyTo(destSS.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeProfilesSize(destSS)
	removedSize := 0
	srcSS.Profiles().RemoveIf(func(srcProfile pprofile.Profile) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rsSize := sz.DeltaSize(sz.ProfileSize(srcProfile))
		if rsSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rsSize
		removedSize += rsSize
		srcProfile.MoveTo(destSS.Profiles().AppendEmpty())
		return true
	})
	return destSS, removedSize
}
