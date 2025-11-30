// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pref // import "go.opentelemetry.io/collector/pdata/xpdata/pref"

import (
	"reflect"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// MarkPipelineOwnedProfiles marks the pprofile.Profiles data as owned by the pipeline, returns true if the data were
// previously not owned by the pipeline, otherwise false.
func MarkPipelineOwnedProfiles(pd pprofile.Profiles) bool {
	return internal.GetProfilesState(internal.ProfilesWrapper(pd)).MarkPipelineOwned()
}

func RefProfiles(pd pprofile.Profiles) {
	if EnableRefCounting.IsEnabled() {
		internal.GetProfilesState(internal.ProfilesWrapper(pd)).Ref()
	}
}

func UnrefProfiles(pd pprofile.Profiles) {
	if EnableRefCounting.IsEnabled() {
		if !internal.GetProfilesState(internal.ProfilesWrapper(pd)).Unref() {
			return
		}
		// Don't call DeleteExportLogsServiceRequest without the gate because we reset the data and that may still cause issues.
		if internal.UseProtoPooling.IsEnabled() {
			internal.DeleteExportProfilesServiceRequest(internal.GetProfilesOrig(internal.ProfilesWrapper(pd)), true)
		}
	}
}

// TODO: Generate this in pdata.

func EqualProfiles(pd1, pd2 pprofile.Profiles) bool {
	return reflect.DeepEqual(internal.GetProfilesOrig(internal.ProfilesWrapper(pd1)), internal.GetProfilesOrig(internal.ProfilesWrapper(pd2)))
}
