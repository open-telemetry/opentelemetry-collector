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

// Package zpages contains the config on setting up ZPages for diagnostics.
package zpages

import (
	"fmt"
	"net"
	"net/http"

	"go.opencensus.io/zpages"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// Run runs a zPages HTTP endpoint with the given config.
func Run(logger *zap.Logger, asyncErrorChannel chan<- error, cfg *configmodels.ZPagesSettings) (closeFn func() error, err error) {
	logger.Info("Setting up zPages...")
	zPagesMux := http.NewServeMux()
	zpages.Handle(zPagesMux, "/debug")

	addr := fmt.Sprintf(":%d", cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to run zPages on %q: %v", addr, err)
	}

	srv := http.Server{Handler: zPagesMux}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			asyncErrorChannel <- fmt.Errorf("failed to server zPages: %v", err)
		}
	}()

	logger.Info("Running zPages", zap.String("address", addr))
	return srv.Close, nil
}
