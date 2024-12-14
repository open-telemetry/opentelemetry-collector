// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererrorprofiles // import "go.opentelemetry.io/collector/consumer/consumererror/consumererrorprofiles"

import "go.opentelemetry.io/collector/consumer/consumererror/xconsumererror"

// Profiles is an error that may carry associated Profile data for a subset of received data
// that failed to be processed or sent.
// Deprecated: [0.116.0] Use xconsumererror.Profiles instead.
type Profiles = xconsumererror.Profiles

// NewProfiles creates a Profiles that can encapsulate received data that failed to be processed or sent.
// Deprecated: [0.116.0] Use xconsumererror.NewProfiles instead.
var NewProfiles = xconsumererror.NewProfiles
