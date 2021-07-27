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

package zpagesextension

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

type zpagesExtension struct {
	config *Config
	logger *zap.Logger
	server http.Server
	stopCh chan struct{}
}

func (zpe *zpagesExtension) Start(_ context.Context, host component.Host) error {
	zPagesMux := http.NewServeMux()

	hostZPages, ok := host.(interface {
		RegisterZPages(mux *http.ServeMux, pathPrefix string)
	})
	if ok {
		zpe.logger.Info("Register Host's zPages")
		hostZPages.RegisterZPages(zPagesMux, "/debug")
	} else {
		zpe.logger.Info("Host's zPages not available")
	}

	// Start the listener here so we can have earlier failure if port is
	// already in use.
	ln, err := zpe.config.TCPAddr.Listen()
	if err != nil {
		return err
	}

	zpe.logger.Info("Starting zPages extension", zap.Any("config", zpe.config))
	zpe.server = http.Server{Handler: zPagesMux}
	zpe.stopCh = make(chan struct{})
	go func() {
		defer close(zpe.stopCh)

		if err := zpe.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			host.ReportFatalError(err)
		}
	}()

	return nil
}

func (zpe *zpagesExtension) Shutdown(context.Context) error {
	err := zpe.server.Close()
	if zpe.stopCh != nil {
		<-zpe.stopCh
	}
	return err
}

func newServer(config *Config, logger *zap.Logger) *zpagesExtension {
	return &zpagesExtension{
		config: config,
		logger: logger,
	}
}
