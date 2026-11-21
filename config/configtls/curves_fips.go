// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import "crypto/tls"

// tlsCurveTypes maps the accepted curve_preferences names to crypto/tls curve IDs.
//
// The requirefips build tag restricts the accepted curves at build time, whether or
// not the process runs in FIPS 140-3 mode. It is kept for builders that already rely
// on it. It is no longer required: the default build detects FIPS 140-3 mode at
// runtime with crypto/fips140.Enabled() (see curves_nofips.go).
var tlsCurveTypes = map[string]tls.CurveID{
	"P256": tls.CurveP256,
	"P384": tls.CurveP384,
	"P521": tls.CurveP521,
}

// defaultCurvePreferences is used when curve_preferences is empty. With the
// requirefips build tag only the curves above are offered.
var defaultCurvePreferences = []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521}
