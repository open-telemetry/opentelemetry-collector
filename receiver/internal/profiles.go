// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/internal"

import "go.opentelemetry.io/collector/component"

// Profiless receiver receives profiles.
// Its purpose is to translate data from any format to the collector's internal profile format.
// ProfilesReceiver feeds a consumer.Profiles with data.
//
// For example, it could be a pprof data source which translates pprof profiles into pprofile.Profiles.
type Profiles interface {
	component.Component
}
