// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"net/http"
	"path"
)

const (
	// Paths
	zPipelinePath = "pipelinez"
)

func (host *serviceHost) RegisterZPages(mux *http.ServeMux, pathPrefix string) {
	mux.HandleFunc(path.Join(pathPrefix, zPipelinePath), host.pipelines.HandleZPages)
}
