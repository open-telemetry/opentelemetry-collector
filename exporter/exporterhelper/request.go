// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

type RequestSizerType = request.SizerType

var (
	RequestSizerTypeBytes    = request.SizerTypeBytes
	RequestSizerTypeItems    = request.SizerTypeItems
	RequestSizerTypeRequests = request.SizerTypeRequests
)
