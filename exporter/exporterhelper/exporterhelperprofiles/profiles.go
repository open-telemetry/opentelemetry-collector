// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] Use go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper instead.
package exporterhelperprofiles // import "go.opentelemetry.io/collector/exporter/exporterhelper/exporterhelperprofiles"

import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

// NewProfilesExporter creates an xexporter.Profiles that records observability metrics and wraps every request with a Span.
// Deprecated: [0.116.0] Use xexporterhelper.NewProfilesExporter instead.
var NewProfilesExporter = xexporterhelper.NewProfilesExporter

// RequestFromProfilesFunc converts pprofile.Profiles into a user-defined Request.
// Deprecated: [0.116.0] Use xexporterhelper.RequestFromProfilesFunc instead.
type RequestFromProfilesFunc = xexporterhelper.RequestFromProfilesFunc

// NewProfilesRequestExporter creates a new profiles exporter based on a custom ProfilesConverter and RequestSender.
// Deprecated: [0.116.0] Use xexporterhelper.NewProfilesRequestExporter instead.
var NewProfilesRequestExporter = xexporterhelper.NewProfilesRequestExporter
