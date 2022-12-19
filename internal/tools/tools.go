// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools
// +build tools

package tools // import "go.opentelemetry.io/collector/internal/tools"

// This file follows the recommendation at
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// on how to pin tooling dependencies to a go.mod file.
// This ensures that all systems use the same version of tools in addition to regular dependencies.

import (
	_ "github.com/a8m/envsubst/cmd/envsubst"
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/addlicense"
	_ "github.com/jcchavezs/porto/cmd/porto"
	_ "github.com/mikefarah/yq/v4"
	_ "github.com/ory/go-acc"
	_ "github.com/pavius/impi/cmd/impi"
	_ "github.com/wadey/gocovmerge"
	_ "go.opentelemetry.io/build-tools/checkdoc"
	_ "go.opentelemetry.io/build-tools/chloggen"
	_ "go.opentelemetry.io/build-tools/crosslink"
	_ "go.opentelemetry.io/build-tools/multimod"
	_ "go.opentelemetry.io/build-tools/semconvgen"
	_ "golang.org/x/exp/cmd/apidiff"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
