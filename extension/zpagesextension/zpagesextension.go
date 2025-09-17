// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension // import "go.opentelemetry.io/collector/extension/zpagesextension"

import (
	"context"
	"errors"
	"expvar"
	"net/http"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/service/hostcapabilities"
)

const (
	expvarzPath = "/debug/expvarz"
)

type zpagesExtension struct {
	config    *Config
	telemetry component.TelemetrySettings
	server    *http.Server
	stopCh    chan struct{}
}

func (zpe *zpagesExtension) Start(ctx context.Context, host component.Host) error {
	var zPagesMux *http.ServeMux
	if zpagesHost, ok := host.(hostcapabilities.ZPages); ok {
		zPagesMux = zpagesHost.GetZPagesMux()
	}
	if zPagesMux == nil {
		zpe.telemetry.Logger.Debug("host does not provide a zPages mux, creating one")
		zPagesMux = http.NewServeMux()
	}

	if zpe.config.Expvar.Enabled {
		zPagesMux.Handle(expvarzPath, expvar.Handler())
		zpe.telemetry.Logger.Info("Registered zPages expvar handler")
	}

	// Start the listener here so we can have earlier failure if port is
	// already in use.
	ln, err := zpe.config.ToListener(ctx)
	if err != nil {
		return err
	}

	zpe.telemetry.Logger.Info("Starting zPages extension", zap.Any("config", zpe.config))
	zpe.server, err = zpe.config.ToServer(ctx, host, zpe.telemetry, zPagesMux)
	if err != nil {
		return err
	}
	zpe.stopCh = make(chan struct{})
	go func() {
		defer close(zpe.stopCh)

		if errHTTP := zpe.server.Serve(ln); errHTTP != nil && !errors.Is(errHTTP, http.ErrServerClosed) {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errHTTP))
		}
	}()

	return nil
}

func (zpe *zpagesExtension) Shutdown(context.Context) error {
	if zpe.server == nil {
		return nil
	}
	err := zpe.server.Close()
	if zpe.stopCh != nil {
		<-zpe.stopCh
	}
	return err
}

func newServer(config *Config, telemetry component.TelemetrySettings) *zpagesExtension {
	return &zpagesExtension{
		config:    config,
		telemetry: telemetry,
	}
}
