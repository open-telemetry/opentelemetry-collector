// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import "embed"

// TemplateFS ensures that the files needed
// to generate metadata as an embedded filesystem since
// `go get` doesn't require these files to be downloaded.
//
//go:embed templates/*.tmpl templates/testdata/*.tmpl
var TemplateFS embed.FS
