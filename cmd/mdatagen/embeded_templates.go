// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import "embed"

// templateFS ensures that the files needed
// to generate metadata as an embedded filesystem since
// `go get` doesn't require these files to be downloaded.
//
//go:embed templates/*.tmpl templates/testdata/*.tmpl
var templateFS embed.FS
