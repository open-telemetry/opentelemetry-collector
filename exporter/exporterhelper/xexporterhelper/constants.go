// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"errors"
)

var (
	// errNilConfig is returned when an empty name is given.
	errNilConfig = errors.New("nil config")
	// errNilLogger is returned when a logger is nil
	errNilLogger = errors.New("nil logger")
	// errNilConsumeRequest is returned when a nil PushTraces is given.
	errNilConsumeRequest = errors.New("nil RequestConsumeFunc")
	// errNilPushProfileData is returned when a nil PushProfiles is given.
	errNilPushProfileData = errors.New("nil PushProfiles")
	// errNilProfilesConverter is returned when a nil RequestFromProfilesFunc is given.
	errNilProfilesConverter = errors.New("nil RequestFromProfilesFunc")
)
