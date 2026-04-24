// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// MergeSplit splits and/or merges the profiles into multiple requests based on the MaxSizeConfig.
//
// Following the OTLP 1.7.0 upgrade, this is currently a noop.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/13106
func (req *profilesRequest) MergeSplit(_ context.Context, maxSizePerSizer map[request.SizerType]int64, r2 request.Request) ([]request.Request, error) {
	sizers := make(map[request.SizerType]sizer.ProfilesSizer)
	for szt := range maxSizePerSizer {
		switch szt {
		case request.SizerTypeItems:
			sizers[szt] = &sizer.ProfilesCountSizer{}
		case request.SizerTypeBytes:
			sizers[szt] = &sizer.ProfilesBytesSizer{}
		default:
			return nil, errors.New("unknown sizer type")
		}
	}

	if r2 != nil && r2.ItemsCount() > 0 {
		req2, ok := r2.(*profilesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		err := req2.mergeTo(req, sizers)
		if err != nil {
			return nil, fmt.Errorf("failed merging profiles; %w", err)
		}
	}

	// If no limits we can simply merge the new request into the current and return.
	if len(maxSizePerSizer) == 0 {
		return []request.Request{req}, nil
	}

	return req.split(maxSizePerSizer, sizers)
}

func (req *profilesRequest) mergeTo(dst *profilesRequest, sizers map[request.SizerType]sizer.ProfilesSizer) error {
	for szt, sz := range sizers {
		dst.setCachedSize(szt, dst.size(szt, sz)+req.size(szt, sz))
		req.setCachedSize(szt, 0)
	}
	return req.pd.MergeTo(dst.pd)
}

func (req *profilesRequest) split(maxSizePerSizer map[request.SizerType]int64, sizers map[request.SizerType]sizer.ProfilesSizer) ([]request.Request, error) {
	var res []request.Request
	sortedSzt := getSortedSizerTypes(maxSizePerSizer)
	for req.pd.SampleCount() > 0 {
		pd := req.pd
		isInitial := true
		exceededAny := false

		for _, szt := range sortedSzt {
			maxSize := maxSizePerSizer[szt]
			sz := sizers[szt]
			if maxSize > 0 && int64(sz.ProfilesSize(pd)) > maxSize {
				exceededAny = true
				pdNew := extractProfiles(pd, int(maxSize), sz)
				if pdNew.SampleCount() == 0 {
					return res, fmt.Errorf("one sample size is greater than max size, dropping items: %d", pd.SampleCount())
				}

				if isInitial {
					isInitial = false
				} else {
					// Prepend remainder to req.pd to maintain order!
					newReqPd := pprofile.NewProfiles()
					pd.ResourceProfiles().MoveAndAppendTo(newReqPd.ResourceProfiles())
					req.pd.ResourceProfiles().MoveAndAppendTo(newReqPd.ResourceProfiles())
					req.pd = newReqPd
				}

				pd = pdNew
			}
		}

		if !exceededAny {
			break
		}

		req.cachedSizes = make(map[request.SizerType]int)
		res = append(res, newProfilesRequest(pd))
	}
	res = append(res, req)
	return res, nil
}

// extractProfiles extracts a new profiles with a maximum number of samples.
func extractProfiles(srcProfiles pprofile.Profiles, capacity int, sz sizer.ProfilesSizer) pprofile.Profiles {
	destProfiles := pprofile.NewProfiles()
	capacityLeft := capacity - sz.ProfilesSize(destProfiles)

	srcProfiles.Dictionary().CopyTo(destProfiles.Dictionary())
	srcProfiles.ResourceProfiles().RemoveIf(func(srcRP pprofile.ResourceProfiles) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRpSize := sz.ResourceProfilesSize(srcRP)
		rpSize := sz.DeltaSize(rawRpSize)

		if rpSize > capacityLeft {
			extSrcRP := extractResourceProfiles(srcRP, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no profiles.
			// Do not add it to the destination if that is the case.
			if extSrcRP.ScopeProfiles().Len() > 0 {
				extSrcRP.MoveTo(destProfiles.ResourceProfiles().AppendEmpty())
			}
			return extSrcRP.ScopeProfiles().Len() != 0
		}
		capacityLeft -= rpSize
		srcRP.MoveTo(destProfiles.ResourceProfiles().AppendEmpty())
		return true
	})
	return destProfiles
}

// extractResourceProfiles extracts profiles and returns a new resource profiles with the specified number of profiles.
func extractResourceProfiles(srcRP pprofile.ResourceProfiles, capacity int, sz sizer.ProfilesSizer) pprofile.ResourceProfiles {
	destRP := pprofile.NewResourceProfiles()
	destRP.SetSchemaUrl(srcRP.SchemaUrl())
	srcRP.Resource().CopyTo(destRP.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceProfilesSize(destRP)
	srcRP.ScopeProfiles().RemoveIf(func(srcSS pprofile.ScopeProfiles) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rawSlSize := sz.ScopeProfilesSize(srcSS)
		ssSize := sz.DeltaSize(rawSlSize)
		if ssSize > capacityLeft {
			extSrcSS := extractScopeProfiles(srcSS, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no profiles.
			// Do not add it to the destination if that is the case.
			if extSrcSS.Profiles().Len() > 0 {
				extSrcSS.MoveTo(destRP.ScopeProfiles().AppendEmpty())
			}
			return extSrcSS.Profiles().Len() != 0
		}
		capacityLeft -= ssSize
		srcSS.MoveTo(destRP.ScopeProfiles().AppendEmpty())
		return true
	})

	return destRP
}

// extractScopeProfiles extracts profiles and returns a new scope profiles with the specified number of profiles.
func extractScopeProfiles(srcSS pprofile.ScopeProfiles, capacity int, sz sizer.ProfilesSizer) pprofile.ScopeProfiles {
	destSS := pprofile.NewScopeProfiles()
	destSS.SetSchemaUrl(srcSS.SchemaUrl())
	srcSS.Scope().CopyTo(destSS.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeProfilesSize(destSS)
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
		srcProfile.MoveTo(destSS.Profiles().AppendEmpty())
		return true
	})
	return destSS
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
