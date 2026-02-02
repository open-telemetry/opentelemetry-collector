// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import "crypto/tls"

var tlsCurveTypes = map[string]tls.CurveID{
	"P256":           tls.CurveP256,
	"P384":           tls.CurveP384,
	"P521":           tls.CurveP521,
	"X25519":         tls.X25519,
	"X25519MLKEM768": tls.X25519MLKEM768,
}

// defaultCurvePreferences defines the default curve preference order when not explicitly configured.
// Hybrid post-quantum KEX (X25519MLKEM768) is preferred, followed by X25519 and NIST curves
// in decreasing order of strength.
var defaultCurvePreferences = []tls.CurveID{
	tls.X25519MLKEM768,
	tls.X25519,
	tls.CurveP521,
	tls.CurveP384,
	tls.CurveP256,
}
