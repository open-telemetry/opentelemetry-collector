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

// This extension implements a proxy to route client requests for sampling config
// to the Jaeger collector.
//
// +---------------+                   +--------------+              +-----------------+
// |               |       get         |              |    proxy     |                 |
// |    client     +---  sampling ---->+    agent     +------------->+    collector    |
// |               |     strategy      |              |              |                 |
// +---------------+                   +--------------+              +-----------------+

package remotesamplingextension

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"

	jAgent "github.com/jaegertracing/jaeger/cmd/agent/app/configmanager/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/open-telemetry/opentelemetry-collector/extension"
)

const mimeTypeApplicationJSON = "application/json"

type remoteSamplingExtension struct {
	config Config
	logger *zap.Logger
	server http.Server
	jProxy *jAgent.SamplingManager
}

func (rs *remoteSamplingExtension) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		services := r.URL.Query()["service"]
		if len(services) != 1 {
			http.Error(w, "'service' parameter must be provided once", http.StatusBadRequest)
		}

		// Jaeger agent's GRPC handler to call the upstream collector.
		resp, err := rs.jProxy.GetSamplingStrategy(services[0])
		if err != nil {
			http.Error(w, fmt.Sprintf("collector error: %+v", err), http.StatusInternalServerError)
			return
		}
		jsonBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Cannot marshall Thrift to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", mimeTypeApplicationJSON)
		if _, err := w.Write(jsonBytes); err != nil {
			return
		}
		return
	})
}

// Start implements ServiceExtension interface
func (rs *remoteSamplingExtension) Start(host extension.Host) error {
	rs.logger.Info("Starting remote sampling extension", zap.Any("config", rs.config))

	// Initialize listener to accept client requests for sampling config
	portStr := ":" + strconv.Itoa(int(rs.config.Port))
	ln, err := net.Listen("tcp", portStr)
	if err != nil {
		host.ReportFatalError(err)
		return nil
	}

	// grpc connection to the upstream Jaeger collector
	conn, err := grpc.Dial(rs.config.Addr, grpc.WithInsecure())
	rs.jProxy = jAgent.NewConfigManager(conn)

	// Register the http handler at the sampling URI
	rs.server.Handler = rs.Handler()

	go func() {
		// The listener ownership goes to the server.
		if err := rs.server.Serve(ln); err != http.ErrServerClosed && err != nil {
			host.ReportFatalError(err)
		}
	}()

	return nil
}

func (rs *remoteSamplingExtension) Shutdown() error {
	return rs.server.Close()
}

func newServer(config Config, logger *zap.Logger) (*remoteSamplingExtension, error) {
	rs := &remoteSamplingExtension{
		config: config,
		logger: logger,
		server: http.Server{},
	}

	return rs, nil
}
