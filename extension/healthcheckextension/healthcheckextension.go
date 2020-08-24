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

package healthcheckextension

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/jaegertracing/jaeger/pkg/healthcheck"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

type healthCheckExtension struct {
	config Config
	logger *zap.Logger
	state  *healthcheck.HealthCheck
	server http.Server
}

var _ component.PipelineWatcher = (*healthCheckExtension)(nil)

func (hc *healthCheckExtension) Start(_ context.Context, host component.Host) error {

	hc.logger.Info("Starting health_check extension", zap.Any("config", hc.config))

	// Initialize listener
	portStr := ":" + strconv.Itoa(int(hc.config.Port))
	ln, err := net.Listen("tcp", portStr)
	if err != nil {
		host.ReportFatalError(err)
		return nil
	}

	// Mount HC handler
	hc.server.Handler = hc.state.Handler()

	go func() {
		// The listener ownership goes to the server.
		if err := hc.server.Serve(ln); err != http.ErrServerClosed && err != nil {
			host.ReportFatalError(err)
		}
	}()

	return nil
}

func (hc *healthCheckExtension) Shutdown(context.Context) error {
	return hc.server.Close()
}

func (hc *healthCheckExtension) Ready() error {
	hc.state.Set(healthcheck.Ready)
	return nil
}

func (hc *healthCheckExtension) NotReady() error {
	hc.state.Set(healthcheck.Unavailable)
	return nil
}

func newServer(config Config, logger *zap.Logger) *healthCheckExtension {
	hc := &healthCheckExtension{
		config: config,
		logger: logger,
		state:  healthcheck.New(),
		server: http.Server{},
	}

	hc.state.SetLogger(logger)

	return hc
}
