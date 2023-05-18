// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	for _, fp := range internal.AllPackages {
		check(fp.GenerateFiles())
		check(fp.GenerateTestFiles())
		check(fp.GenerateInternalFiles())
	}
}
