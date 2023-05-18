// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components // import "go.opentelemetry.io/collector/service/internal/components"

import (
	"net/http"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

// hostWrapper adds behavior on top of the component.Host being passed when starting the built components.
type hostWrapper struct {
	component.Host
	*zap.Logger
}

func NewHostWrapper(host component.Host, logger *zap.Logger) component.Host {
	return &hostWrapper{
		host,
		logger,
	}
}

func (hw *hostWrapper) ReportFatalError(err error) {
	// The logger from the built component already identifies the component.
	hw.Logger.Error("Component fatal error", zap.Error(err))
	hw.Host.ReportFatalError(err)
}

// RegisterZPages is used by zpages extension to register handles from service.
// When the wrapper is passed to the extension it won't be successful when casting
// the interface, for the time being expose the interface here.
// TODO: Find a better way to add the service zpages to the extension. This a temporary fix.
func (hw *hostWrapper) RegisterZPages(mux *http.ServeMux, pathPrefix string) {
	if zpagesHost, ok := hw.Host.(interface {
		RegisterZPages(mux *http.ServeMux, pathPrefix string)
	}); ok {
		zpagesHost.RegisterZPages(mux, pathPrefix)
	}
}
