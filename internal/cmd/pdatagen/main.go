// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	for _, fp := range pdata.AllPackages {
		check(fp.GenerateFiles())
		check(fp.GenerateTestFiles())
		check(fp.GenerateInternalFiles())
		check(fp.GenerateInternalTestsFiles())
	}
}
