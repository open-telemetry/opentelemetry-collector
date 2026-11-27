// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import "unicode/utf8"

func validUTF8ProfilesDictionary(d *ProfilesDictionary) bool {
	if !validUTF8StringSlice(d.StringTable) {
		return false
	}
	for _, kv := range d.AttributeTable {
		if kv != nil && !validUTF8AnyValue(&kv.Value) {
			return false
		}
	}
	return true
}

// ValidateUTF8ExportProfilesServiceRequest returns false when any string in req contains invalid UTF-8.
func ValidateUTF8ExportProfilesServiceRequest(req *ExportProfilesServiceRequest) bool {
	if !validUTF8ProfilesDictionary(&req.Dictionary) {
		return false
	}
	for _, rp := range req.ResourceProfiles {
		if !validUTF8Resource(&rp.Resource) || !utf8.ValidString(rp.SchemaUrl) {
			return false
		}
		for _, sp := range rp.ScopeProfiles {
			if !validUTF8InstrumentationScope(&sp.Scope) || !utf8.ValidString(sp.SchemaUrl) {
				return false
			}
			for _, p := range sp.Profiles {
				if !utf8.ValidString(p.OriginalPayloadFormat) {
					return false
				}
			}
		}
	}
	return true
}

// RejectInvalidUTF8ExportProfilesServiceRequest removes profiles containing invalid UTF-8 from req
// and returns the number removed. When the shared dictionary is invalid every profile is removed.
func RejectInvalidUTF8ExportProfilesServiceRequest(req *ExportProfilesServiceRequest) int {
	rejected := 0
	if !validUTF8ProfilesDictionary(&req.Dictionary) {
		for _, rp := range req.ResourceProfiles {
			for _, sp := range rp.ScopeProfiles {
				rejected += len(sp.Profiles)
			}
		}
		req.ResourceProfiles = removeIf(req.ResourceProfiles, DeleteResourceProfiles, func(*ResourceProfiles) bool { return true })
		return rejected
	}
	req.ResourceProfiles = removeIf(req.ResourceProfiles, DeleteResourceProfiles, func(rp *ResourceProfiles) bool {
		if !validUTF8Resource(&rp.Resource) || !utf8.ValidString(rp.SchemaUrl) {
			for _, sp := range rp.ScopeProfiles {
				rejected += len(sp.Profiles)
			}
			return true
		}
		if len(rp.ScopeProfiles) == 0 {
			return false
		}
		rp.ScopeProfiles = removeIf(rp.ScopeProfiles, DeleteScopeProfiles, func(sp *ScopeProfiles) bool {
			if !validUTF8InstrumentationScope(&sp.Scope) || !utf8.ValidString(sp.SchemaUrl) {
				rejected += len(sp.Profiles)
				return true
			}
			if len(sp.Profiles) == 0 {
				return false
			}
			sp.Profiles = removeIf(sp.Profiles, DeleteProfile, func(p *Profile) bool {
				invalid := !utf8.ValidString(p.OriginalPayloadFormat)
				if invalid {
					rejected++
				}
				return invalid
			})
			return len(sp.Profiles) == 0
		})
		return len(rp.ScopeProfiles) == 0
	})
	return rejected
}
