// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import "crypto/tls"

// tlsCurveTypes maps the accepted curve_preferences names to crypto/tls curve IDs.
//
// This is the default build. FIPS 140-3 mode is detected at runtime with
// crypto/fips140.Enabled() (GODEBUG=fips140=on or only, or a binary built with
// GOFIPS140): X25519 is rejected when the configuration is validated and crypto/tls
// filters the remaining curves. The requirefips build tag is not needed for this.
var tlsCurveTypes = map[string]tls.CurveID{
	"P256":               tls.CurveP256,
	"P384":               tls.CurveP384,
	"P521":               tls.CurveP521,
	"X25519":             tls.X25519,
	"X25519MLKEM768":     tls.X25519MLKEM768,
	"SecP256r1MLKEM768":  tls.SecP256r1MLKEM768,
	"SecP384r1MLKEM1024": tls.SecP384r1MLKEM1024,
}

// defaultCurvePreferences is used when curve_preferences is empty. nil lets crypto/tls
// apply its own defaults, which it already filters in FIPS 140-3 mode.
var defaultCurvePreferences []tls.CurveID
