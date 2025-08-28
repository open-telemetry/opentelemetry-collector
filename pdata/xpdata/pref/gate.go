// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pref // import "go.opentelemetry.io/collector/pdata/xpdata/pref"

import (
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal"
)

// UseProtoPooling temporary expose public to allow testing.
var UseProtoPooling = internal.UseProtoPooling

var EnableRefCounting = featuregate.GlobalRegistry().MustRegister(
	"pdata.enableRefCounting",
	featuregate.StageBeta,
	featuregate.WithRegisterDescription("When enabled, enables using ref counting to know when pdata memory can be freed up. This featuregate is here only to protect if unexpected bugs happens because of ref counting logic."),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/issues/13631"),
	featuregate.WithRegisterFromVersion("v0.133.0"),
)
