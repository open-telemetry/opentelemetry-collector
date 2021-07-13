// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awsecshealthcheckextension

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/jaegertracing/jaeger/pkg/healthcheck"
	"go.uber.org/zap"

	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

type ECSHealthCheckExtension struct {
	config   Config
	logger   *zap.Logger
	state    *healthcheck.HealthCheck
	server   http.Server
	stopCh   chan struct{}
	exporter *ECSHealthCheckExporter
}

// initViews function could register the "failed to send" view and the customized exporter
func (hc *ECSHealthCheckExtension) initViews() error {
	if err := view.Register(&view.View{
		Name:        "AOC/ECSHealthCheckExporterFailedToSendSpans",
		Description: "number of errors in exporters over span",
		Measure:     obsmetrics.ExporterFailedToSendSpans,
		Aggregation: view.Count(),
	}); err != nil {
		log.Fatalf("Failed to register view: %v", err)
		return err
	}

	hc.exporter = newECSHealthCheckExporter()
	view.RegisterExporter(hc.exporter)
	return nil
}

func (hc *ECSHealthCheckExtension) Start(_ context.Context, host component.Host) error {
	hc.logger.Info("Starting ECS health check extension", zap.Any("config", hc.config))

	// Initialize listener
	var (
		ln  net.Listener
		err error
	)

	err = hc.initViews()
	if err != nil {
		return err
	}

	err = hc.config.Validate()
	if err != nil {
		return err
	}

	if hc.config.TCPAddr.Endpoint == "" || hc.config.TCPAddr.Endpoint == defaultEndpoint {
		ln, err = net.Listen("tcp", defaultEndpoint)
	} else {
		ln, err = hc.config.TCPAddr.Listen()
	}
	if err != nil {
		return err
	}

	hc.server.Handler = hc.handler()
	hc.stopCh = make(chan struct{})
	go func() {
		defer close(hc.stopCh)

		if err := hc.server.Serve(ln); err != http.ErrServerClosed && err != nil {
			host.ReportFatalError(err)
		}
	}()
	return nil
}

func (hc *ECSHealthCheckExtension) check() bool {
	hc.exporter.mu.Lock()
	defer hc.exporter.mu.Unlock()

	interval, _ := time.ParseDuration(hc.config.Interval)
	hc.exporter.rotate(interval)

	return hc.config.ExporterErrorLimit >= len(hc.exporter.exporterErrorQueue)
}

func (hc *ECSHealthCheckExtension) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hc.check() {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(strconv.Itoa(len(hc.exporter.exporterErrorQueue))))
		} else {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(strconv.Itoa(len(hc.exporter.exporterErrorQueue))))
		}
	})
}

func (hc *ECSHealthCheckExtension) Shutdown(context.Context) error {
	err := hc.server.Close()
	if hc.stopCh != nil {
		<-hc.stopCh
	}
	return err
}

func newServer(config Config, logger *zap.Logger) *ECSHealthCheckExtension {
	hc := &ECSHealthCheckExtension{
		config:   config,
		logger:   logger,
		state:    healthcheck.New(),
		server:   http.Server{},
		exporter: newECSHealthCheckExporter(),
	}

	hc.state.SetLogger(logger)

	return hc
}
