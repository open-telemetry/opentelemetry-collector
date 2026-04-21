// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pref // import "go.opentelemetry.io/collector/pdata/xpdata/pref"

import (
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

// UseProtoPooling temporary expose public to allow testing.
var UseProtoPooling = metadata.PdataUseProtoPoolingFeatureGate
