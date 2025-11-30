// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"
)

// checkErr prints the given error and exits when e is non-nil.
func checkErr(e error) {
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
}

func main() {
	var workdir string
	flag.StringVar(&workdir, "C", ".", "set work directory")
	flag.Parse()

	checkErr(os.Chdir(workdir))
	checkErr(pdata.DeleteGeneratedFiles(filepath.Join("pdata", "internal")))
	for _, fp := range pdata.AllPackages {
		checkErr(pdata.DeleteGeneratedFiles(filepath.Join("pdata", fp.Path())))
		checkErr(fp.GenerateFiles())
		checkErr(fp.GenerateTestFiles())
		checkErr(fp.GenerateInternalFiles())
		checkErr(fp.GenerateProtoMessageFiles())
		checkErr(fp.GenerateProtoMessageTestsFiles())
		checkErr(fp.GenerateProtoEnumFiles())
	}
}
