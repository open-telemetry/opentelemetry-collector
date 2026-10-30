// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import "crypto/tls"

var tlsCurveTypes = map[string]tls.CurveID{
	"P256":               tls.CurveP256,
	"P384":               tls.CurveP384,
	"P521":               tls.CurveP521,
	"X25519":             tls.X25519,
	"X25519MLKEM768":     tls.X25519MLKEM768,
	"SecP256r1MLKEM768":  tls.SecP256r1MLKEM768,
	"SecP384r1MLKEM1024": tls.SecP384r1MLKEM1024,
}

const requireFIPSBuild = false
