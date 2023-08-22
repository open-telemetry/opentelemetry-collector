// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components // import "go.opentelemetry.io/collector/service/internal/components"

import (
	"net/http"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/servicehost"
	"go.opentelemetry.io/collector/service/internal/status"
)

// hostWrapper adds behavior on top of the component.Host being passed when starting the built components.
type hostWrapper struct {
	servicehost.Host
	component      *component.InstanceID
	statusNotifier status.Notifier
	*zap.Logger
}

func NewHostWrapper(host servicehost.Host, instanceID *component.InstanceID, logger *zap.Logger) component.Host {
	return &hostWrapper{
		host,
		instanceID,
		status.NewNotifier(host, instanceID),
		logger,
	}
}

func (hw *hostWrapper) ReportFatalError(err error) {
	// The logger from the built component already identifies the component.
	hw.Logger.Error("Component fatal error", zap.Error(err))
	hw.Host.ReportFatalError(err) // nolint:staticcheck
}

func (hw *hostWrapper) ReportComponentStatus(status component.Status, options ...component.StatusEventOption) {
	// The following can return an error. The two cases that would result in an error would be:
	//   - An invalid state transition
	//   - Invalid arguments (basically providing a component.WithError option to a non-error status)
	// The latter is a programming error and should be corrected. The former, is something that is
	// likely to happen, but not something the programmer should be concerned about. An example would be
	// reporting StatusRecoverableError multiple times, which, could happen while recovering, however,
	// only the first invocation would result in a successful status transition.
	_ = hw.statusNotifier.Event(status, options...)
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
