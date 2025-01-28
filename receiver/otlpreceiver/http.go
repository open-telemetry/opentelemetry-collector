// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
)

type handler struct {
	pattern     string
	handlerFunc http.HandlerFunc
}

type httpComponent struct {
	cfg        *confighttp.ServerConfig
	serverHTTP *http.Server
	tel        component.TelemetrySettings
	shutdownWG sync.WaitGroup
	handlers   []handler
}

func newSharedHTTP(cfg *confighttp.ServerConfig, tel component.TelemetrySettings) (*sharedcomponent.Component[*httpComponent], error) {
	return https.LoadOrStore(
		cfg,
		func() (*httpComponent, error) {
			return &httpComponent{cfg: cfg, tel: tel}, nil
		},
	)
}

func (r *httpComponent) Start(ctx context.Context, host component.Host) error {
	httpMux := http.NewServeMux()
	for _, h := range r.handlers {
		httpMux.HandleFunc(h.pattern, h.handlerFunc)
	}

	var err error
	if r.serverHTTP, err = r.cfg.ToServer(ctx, host, r.tel, httpMux, confighttp.WithErrorHandler(errorHandler)); err != nil {
		return err
	}

	r.tel.Logger.Info("Starting HTTP server", zap.String("endpoint", r.cfg.Endpoint))
	var hln net.Listener
	if hln, err = r.cfg.ToListener(ctx); err != nil {
		return err
	}

	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()
		if errHTTP := r.serverHTTP.Serve(hln); errHTTP != nil && !errors.Is(errHTTP, http.ErrServerClosed) {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errHTTP))
		}
	}()
	return nil
}

// Shutdown is a method to turn off receiving.
func (r *httpComponent) Shutdown(ctx context.Context) error {
	var err error
	if r.serverHTTP != nil {
		err = r.serverHTTP.Shutdown(ctx)
	}
	r.shutdownWG.Wait()
	return err
}

func (r *httpComponent) registerHandler(pattern string, handlerFunc http.HandlerFunc) {
	r.handlers = append(r.handlers, handler{pattern: pattern, handlerFunc: handlerFunc})
}

// This is the map of already created OTLP receivers for particular configurations.
// We maintain this map because the receiver.Factory is asked trace and metric receivers separately
// when it gets CreateTraces() and CreateMetrics() but they must not
// create separate objects, they must use one otlpReceiver object per configuration.
// When the receiver is shutdown it should be removed from this map so the same configuration
// can be recreated successfully.
var https = sharedcomponent.NewMap[*confighttp.ServerConfig, *httpComponent]()
