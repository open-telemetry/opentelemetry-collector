// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// MergeSplit splits and/or merges the profiles into multiple requests based on the MaxSizeConfig.
func (req *profilesRequest) MergeSplit(_ context.Context, maxSize int, _ exporterhelper.RequestSizerType, r2 exporterhelper.Request) ([]exporterhelper.Request, error) {
	if r2 != nil {
		req2, ok := r2.(*profilesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req)
	}

	// If no limit we can simply merge the new request into the current and return.
	if maxSize == 0 {
		return []exporterhelper.Request{req}, nil
	}
	return req.split(maxSize)
}

func (req *profilesRequest) mergeTo(dst *profilesRequest) {
	dst.setCachedItemsCount(dst.ItemsCount() + req.ItemsCount())
	req.setCachedItemsCount(0)
	req.pd.ResourceProfiles().MoveAndAppendTo(dst.pd.ResourceProfiles())
}

func (req *profilesRequest) split(maxSize int) ([]exporterhelper.Request, error) {
	var res []exporterhelper.Request
	for req.ItemsCount() > maxSize {
		pd := extractProfiles(req.pd, maxSize)
		size := pd.SampleCount()
		req.setCachedItemsCount(req.ItemsCount() - size)
		res = append(res, &profilesRequest{pd: pd, cachedItemsCount: size})
	}
	res = append(res, req)
	return res, nil
}

// extractProfiles extracts a new profiles with a maximum number of samples.
func extractProfiles(srcProfiles pprofile.Profiles, count int) pprofile.Profiles {
	destProfiles := pprofile.NewProfiles()
	srcProfiles.ResourceProfiles().RemoveIf(func(srcRS pprofile.ResourceProfiles) bool {
		if count == 0 {
			return false
		}
		needToExtract := samplesCount(srcRS) > count
		if needToExtract {
			srcRS = extractResourceProfiles(srcRS, count)
		}
		count -= samplesCount(srcRS)
		srcRS.MoveTo(destProfiles.ResourceProfiles().AppendEmpty())
		return !needToExtract
	})
	return destProfiles
}

// extractResourceProfiles extracts profiles and returns a new resource profiles with the specified number of profiles.
func extractResourceProfiles(srcRS pprofile.ResourceProfiles, count int) pprofile.ResourceProfiles {
	destRS := pprofile.NewResourceProfiles()
	destRS.SetSchemaUrl(srcRS.SchemaUrl())
	srcRS.Resource().CopyTo(destRS.Resource())
	srcRS.ScopeProfiles().RemoveIf(func(srcSS pprofile.ScopeProfiles) bool {
		if count == 0 {
			return false
		}
		needToExtract := srcSS.Profiles().Len() > count
		if needToExtract {
			srcSS = extractScopeProfiles(srcSS, count)
		}
		count -= srcSS.Profiles().Len()
		srcSS.MoveTo(destRS.ScopeProfiles().AppendEmpty())
		return !needToExtract
	})
	srcRS.Resource().CopyTo(destRS.Resource())
	return destRS
}

// extractScopeProfiles extracts profiles and returns a new scope profiles with the specified number of profiles.
func extractScopeProfiles(srcSS pprofile.ScopeProfiles, count int) pprofile.ScopeProfiles {
	destSS := pprofile.NewScopeProfiles()
	destSS.SetSchemaUrl(srcSS.SchemaUrl())
	srcSS.Scope().CopyTo(destSS.Scope())
	srcSS.Profiles().RemoveIf(func(srcProfile pprofile.Profile) bool {
		if count == 0 {
			return false
		}
		srcProfile.MoveTo(destSS.Profiles().AppendEmpty())
		count--
		return true
	})
	return destSS
}

// resourceProfilessCount calculates the total number of profiles in the pdata.ResourceProfiles.
func samplesCount(rs pprofile.ResourceProfiles) int {
	count := 0
	rs.ScopeProfiles().RemoveIf(func(ss pprofile.ScopeProfiles) bool {
		ss.Profiles().RemoveIf(func(sp pprofile.Profile) bool {
			count += sp.Sample().Len()
			return false
		})
		return false
	})
	return count
}
