// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher // import "go.opentelemetry.io/collector/exporter/exporterbatcher"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Deprecated: [v0.123.0] use exporterhelper.BatcherConfig
type Config = exporterhelper.BatcherConfig //nolint:staticcheck

// Deprecated: [v0.123.0] use exporterhelper.SizeConfig
type SizeConfig = exporterhelper.SizeConfig //nolint:staticcheck

// Deprecated: [v0.123.0] use exporterhelper.RequestSizerType
type SizerType = exporterhelper.RequestSizerType

// Deprecated: [v0.123.0] use exporterhelper.RequestSizerTypeRequests
var SizerTypeRequests = exporterhelper.RequestSizerTypeRequests

// Deprecated: [v0.123.0] use exporterhelper.RequestSizerTypeItems
var SizerTypeItems = exporterhelper.RequestSizerTypeItems

// Deprecated: [v0.123.0] use exporterhelper.RequestSizerTypeRequests
var SizerTypeBytes = exporterhelper.RequestSizerTypeBytes

// Deprecated: [v0.123.0] use exporterhelper.NewDefaultBatcherConfig
var NewDefaultConfig = exporterhelper.NewDefaultBatcherConfig //nolint:staticcheck
