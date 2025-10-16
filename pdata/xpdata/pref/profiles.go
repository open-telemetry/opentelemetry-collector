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
	return internal.GetProfilesState(internal.Profiles(pd)).MarkPipelineOwned()
}

func RefProfiles(pd pprofile.Profiles) {
	if EnableRefCounting.IsEnabled() {
		internal.GetProfilesState(internal.Profiles(pd)).Ref()
	}
}

func UnrefProfiles(pd pprofile.Profiles) {
	if EnableRefCounting.IsEnabled() {
		if !internal.GetProfilesState(internal.Profiles(pd)).Unref() {
			return
		}
		// Don't call DeleteOrigExportLogsServiceRequest without the gate because we reset the data and that may still cause issues.
		if internal.UseProtoPooling.IsEnabled() {
			internal.DeleteOrigExportProfilesServiceRequest(internal.GetOrigProfiles(internal.Profiles(pd)), true)
		}
	}
}

// TODO: Generate this in pdata.

func EqualProfiles(pd1, pd2 pprofile.Profiles) bool {
	return reflect.DeepEqual(internal.GetOrigProfiles(internal.Profiles(pd1)), internal.GetOrigProfiles(internal.Profiles(pd2)))
}
