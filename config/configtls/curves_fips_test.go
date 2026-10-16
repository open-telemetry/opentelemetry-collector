// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build requirefips

package configtls

// requireFIPSBuild reports whether this binary was built with the requirefips
// tag, which restricts the configurable curves to the FIPS-approved set.
const requireFIPSBuild = true
