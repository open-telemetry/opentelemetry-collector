// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// MergeSplit splits and/or merges the profiles into multiple requests based on the MaxSizeConfig.
//
// Following the OTLP 1.7.0 upgrade, this is currently a noop.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/13106
func (req *profilesRequest) MergeSplit(_ context.Context, maxSize int, szt exporterhelper.RequestSizerType, r2 Request) ([]Request, error) {
	var sz sizer.ProfilesSizer
	switch szt {
	case exporterhelper.RequestSizerTypeItems:
		sz = &sizer.ProfilesSamplesCountSizer{}
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
		err := req2.mergeTo(req, sz)
		if err != nil {
			return nil, fmt.Errorf("failed merging profiles; %w", err)
		}
	}

	// If no limit we can simply merge the new request into the current and return.
	if maxSize == 0 {
		return []Request{req}, nil
	}
	return req.split(maxSize, sz)
}

func (req *profilesRequest) mergeTo(dst *profilesRequest, sz sizer.ProfilesSizer) error {
	if sz != nil {
		dst.setCachedSize(dst.size(sz) + req.size(sz))
		req.setCachedSize(0)
	}
	return req.pd.MergeTo(dst.pd)
}

func (req *profilesRequest) split(maxSize int, sz sizer.ProfilesSizer) ([]Request, error) {
	if req.size(sz) <= maxSize {
		return []Request{req}, nil
	}
	var res []Request
	droppedItems := 0
	for req.size(sz) > maxSize {
		samplesBefore := req.pd.SampleCount()
		pd, removedSize := extractProfiles(req.pd, maxSize, sz)
		if removedSize == 0 {
			// Nothing left the source, so no progress is possible. Stop rather than loop.
			return res, fmt.Errorf("request size is greater than max size and cannot be split further, dropping items: %d", droppedItems+req.pd.SampleCount())
		}
		req.setCachedSize(req.size(sz) - removedSize)
		droppedItems += samplesBefore - req.pd.SampleCount() - pd.SampleCount()
		if pd.SampleCount() > 0 {
			res = append(res, newProfilesRequest(pd))
		}
	}
	// Splitting can leave nothing to export once oversized profiles and sample-less resources are gone.
	if req.pd.SampleCount() > 0 {
		res = append(res, req)
	}
	if droppedItems > 0 {
		return res, fmt.Errorf("single profile exceeds the max size limit, dropping items: %d", droppedItems)
	}
	return res, nil
}

// extractProfiles extracts a new profiles with a maximum number of samples.
func extractProfiles(srcProfiles pprofile.Profiles, capacity int, sz sizer.ProfilesSizer) (pprofile.Profiles, int) {
	destProfiles := pprofile.NewProfiles()
	capacityLeft := capacity - sz.ProfilesSize(destProfiles)
	removedSize := 0

	srcProfiles.Dictionary().CopyTo(destProfiles.Dictionary())
	srcProfiles.ResourceProfiles().RemoveIf(func(srcRP pprofile.ResourceProfiles) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRpSize := sz.ResourceProfilesSize(srcRP)
		rpSize := sz.DeltaSize(rawRpSize)

		if rpSize > capacityLeft {
			extSrcRP, extRpSize := extractResourceProfiles(srcRP, capacityLeft, capacity, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no profiles.
			// Do not add it to the destination if that is the case.
			if extSrcRP.ScopeProfiles().Len() > 0 {
				extSrcRP.MoveTo(destProfiles.ResourceProfiles().AppendEmpty())
			}
			if srcRP.ScopeProfiles().Len() == 0 {
				// Nothing is left in the source resource, so all of it is removed.
				removedSize += rpSize
				return true
			}
			// The source resource shrinks to the delta size of what is left in it.
			removedSize += rpSize - sz.DeltaSize(rawRpSize-extRpSize)
			return false
		}
		capacityLeft -= rpSize
		removedSize += rpSize
		srcRP.MoveTo(destProfiles.ResourceProfiles().AppendEmpty())
		return true
	})
	return destProfiles, removedSize
}

// extractResourceProfiles extracts profiles and returns a new resource profiles with the specified number of profiles.
func extractResourceProfiles(srcRP pprofile.ResourceProfiles, capacity, maxSize int, sz sizer.ProfilesSizer) (pprofile.ResourceProfiles, int) {
	destRP := pprofile.NewResourceProfiles()
	destRP.SetSchemaUrl(srcRP.SchemaUrl())
	srcRP.Resource().CopyTo(destRP.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceProfilesSize(destRP)
	// Room for a scope in an otherwise empty batch, once this resource's header and attributes are paid for.
	maxScopeSize := maxSize - (sz.DeltaSize(maxSize) - maxSize) - sz.ResourceProfilesSize(destRP)
	removedSize := 0

	srcRP.ScopeProfiles().RemoveIf(func(srcSP pprofile.ScopeProfiles) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rawSpSize := sz.ScopeProfilesSize(srcSP)
		spSize := sz.DeltaSize(rawSpSize)
		if spSize > capacityLeft {
			extSrcSP, extSpSize := extractScopeProfiles(srcSP, capacityLeft, maxScopeSize, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no profiles.
			// Do not add it to the destination if that is the case.
			if extSrcSP.Profiles().Len() > 0 {
				extSrcSP.MoveTo(destRP.ScopeProfiles().AppendEmpty())
			}
			if srcSP.Profiles().Len() == 0 {
				// Nothing is left in the source scope, so all of it is removed.
				removedSize += spSize
				return true
			}
			// The source scope shrinks to the delta size of what is left in it.
			removedSize += spSize - sz.DeltaSize(rawSpSize-extSpSize)
			return false
		}
		capacityLeft -= spSize
		removedSize += spSize
		srcSP.MoveTo(destRP.ScopeProfiles().AppendEmpty())
		return true
	})

	return destRP, removedSize
}

// extractScopeProfiles extracts profiles and returns a new scope profiles with the specified number of profiles.
func extractScopeProfiles(srcSP pprofile.ScopeProfiles, capacity, maxScopeSize int, sz sizer.ProfilesSizer) (pprofile.ScopeProfiles, int) {
	destSP := pprofile.NewScopeProfiles()
	destSP.SetSchemaUrl(srcSP.SchemaUrl())
	srcSP.Scope().CopyTo(destSP.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeProfilesSize(destSP)
	// Largest profile that fits an otherwise empty batch, once the resource and scope headers and attributes are paid for.
	maxProfileSize := maxScopeSize - (sz.DeltaSize(maxScopeSize) - maxScopeSize) - sz.ScopeProfilesSize(destSP)
	removedSize := 0
	srcSP.Profiles().RemoveIf(func(srcProfile pprofile.Profile) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		profileSize := sz.DeltaSize(sz.ProfileSize(srcProfile))
		if profileSize > maxProfileSize {
			// It can never be exported and would block every profile behind it, so drop it.
			removedSize += profileSize
			return true
		}
		if profileSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= profileSize
		removedSize += profileSize
		srcProfile.MoveTo(destSP.Profiles().AppendEmpty())
		return true
	})
	return destSP, removedSize
}
