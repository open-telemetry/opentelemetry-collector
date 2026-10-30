// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import "crypto/tls"

// This mirrors allowedCurvePreferencesFIPS in crypto/tls/defaults_fips140.go,
// the filter crypto/tls applies when FIPS-approved settings are required. The
// hybrid key exchanges are approved there, so only bare X25519 is missing
// relative to the non-FIPS map.
// See https://cs.opensource.google/go/go/+/refs/tags/go1.26.0:src/crypto/tls/defaults_fips140.go
var tlsCurveTypes = map[string]tls.CurveID{
	"P256":               tls.CurveP256,
	"P384":               tls.CurveP384,
	"P521":               tls.CurveP521,
	"X25519MLKEM768":     tls.X25519MLKEM768,
	"SecP256r1MLKEM768":  tls.SecP256r1MLKEM768,
	"SecP384r1MLKEM1024": tls.SecP384r1MLKEM1024,
}

const requireFIPSBuild = true
