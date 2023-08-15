// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components // import "go.opentelemetry.io/collector/service/internal/components"

import (
	"net/http"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/servicehost"
)

// hostWrapper adds behavior on top of the component.Host being passed when starting the built components.
// TODO: rename this to componentHost or hostComponentConnector to better reflect the purpose.
type hostWrapper struct {
	servicehost.Host
	component *component.InstanceID
	*zap.Logger
}

func NewHostWrapper(host servicehost.Host, component *component.InstanceID, logger *zap.Logger) component.Host {
	return &hostWrapper{
		host,
		component,
		logger,
	}
}

func (hw *hostWrapper) ReportFatalError(err error) {
	// The logger from the built component already identifies the component.
	hw.Logger.Error("Component fatal error", zap.Error(err))
	hw.Host.ReportFatalError(err) // nolint:staticcheck
}

func (hw *hostWrapper) ReportComponentStatus(event *component.StatusEvent) {
	hw.Host.ReportComponentStatus(hw.component, event)
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
