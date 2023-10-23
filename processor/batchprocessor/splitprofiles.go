// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// splitProfiles removes profilerecords from the input data and returns a new data of the specified size.
func splitProfiles(size int, src pprofile.Profiles) pprofile.Profiles {
	if src.ProfileCount() <= size {
		return src
	}
	totalCopiedProfiles := 0
	dest := pprofile.NewProfiles()

	src.ResourceProfiles().RemoveIf(func(srcRl pprofile.ResourceProfiles) bool {
		// If we are done skip everything else.
		if totalCopiedProfiles == size {
			return false
		}

		// If it fully fits
		srcRlLRC := resourceLRCProfiles(srcRl)
		if (totalCopiedProfiles + srcRlLRC) <= size {
			totalCopiedProfiles += srcRlLRC
			srcRl.MoveTo(dest.ResourceProfiles().AppendEmpty())
			return true
		}

		destRl := dest.ResourceProfiles().AppendEmpty()
		srcRl.Resource().CopyTo(destRl.Resource())
		srcRl.ScopeProfiles().RemoveIf(func(srcIll pprofile.ScopeProfiles) bool {
			// If we are done skip everything else.
			if totalCopiedProfiles == size {
				return false
			}

			// If possible to move all metrics do that.
			srcIllLRC := srcIll.Profiles().Len()
			if size >= srcIllLRC+totalCopiedProfiles {
				totalCopiedProfiles += srcIllLRC
				srcIll.MoveTo(destRl.ScopeProfiles().AppendEmpty())
				return true
			}

			destIll := destRl.ScopeProfiles().AppendEmpty()
			srcIll.Scope().CopyTo(destIll.Scope())
			srcIll.Profiles().RemoveIf(func(srcMetric pprofile.Profile) bool {
				// If we are done skip everything else.
				if totalCopiedProfiles == size {
					return false
				}
				srcMetric.MoveTo(destIll.Profiles().AppendEmpty())
				totalCopiedProfiles++
				return true
			})
			return false
		})
		return srcRl.ScopeProfiles().Len() == 0
	})

	return dest
}

// resourceLRCProfiles calculates the total number of profile records in the pprofile.ResourceProfiles.
func resourceLRCProfiles(rs pprofile.ResourceProfiles) (count int) {
	for k := 0; k < rs.ScopeProfiles().Len(); k++ {
		count += rs.ScopeProfiles().At(k).Profiles().Len()
	}
	return
}
