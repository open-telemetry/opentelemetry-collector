// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] Use go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper instead.
package processorhelperprofiles // import "go.opentelemetry.io/collector/processor/processorhelper/processorhelperprofiles"

import (
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
)

// ProcessProfilesFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
// Deprecated: [0.116.0] Use xprocessorhelper.ProcessProfilesFunc instead.
type ProcessProfilesFunc = xprocessorhelper.ProcessProfilesFunc

// NewProfiles creates a xprocessor.Profiles that ensure context propagation.
// Deprecated: [0.116.0] Use xprocessorhelper.NewProfiles instead.
var NewProfiles = xprocessorhelper.NewProfiles
