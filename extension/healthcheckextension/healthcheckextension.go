// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package healthcheckextension

import (
	"net/http"

	"github.com/jaegertracing/jaeger/pkg/healthcheck"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/extension"
)

type healthCheckExtension struct {
	config Config
	logger *zap.Logger
	server *healthcheck.HealthCheck
}

var _ (extension.ServiceExtension) = (*healthCheckExtension)(nil)
var _ (extension.PipelineWatcher) = (*healthCheckExtension)(nil)

func (hc *healthCheckExtension) Start(host extension.Host) error {

	hc.logger.Info("Starting health_check extension", zap.Any("config", hc.config))
	go func() {
		// The listener ownership goes to the server.
		if _, err := hc.server.Serve(int(hc.config.Port)); err != http.ErrServerClosed && err != nil {
			host.ReportFatalError(err)
		}
	}()

	return nil
}

func (hc *healthCheckExtension) Shutdown() error {
	return nil
}

func (hc *healthCheckExtension) Ready() error {
	hc.server.Set(healthcheck.Ready)
	return nil
}

func (hc *healthCheckExtension) NotReady() error {
	hc.server.Set(healthcheck.Unavailable)
	return nil
}

func newServer(config Config, logger *zap.Logger) (*healthCheckExtension, error) {
	hc := &healthCheckExtension{
		config: config,
		logger: logger,
		server: healthcheck.New(healthcheck.Unavailable, healthcheck.Logger(logger)),
	}

	return hc, nil
}
