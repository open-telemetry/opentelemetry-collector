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
	"flag"
	"fmt"
	"net"
	"net/http"

	"go.opencensus.io/zpages"
)

const (
	// ZPagesHTTPPort is the name of the flag used to specify the zpages port.
	// TODO(ccaraman): Move ZPage configuration to be apart of global config/config.go, maybe under diagnostics section.
	ZPagesHTTPPort = "zpages-http-port"
)

// AddFlags adds to the flag set a flag to configure the zpages server.
func AddFlags(flags *flag.FlagSet) {
	flags.Uint(
		ZPagesHTTPPort,
		55679,
		"Port on which to run the zpages http server, use 0 to disable zpages.")
}

// Run run a zPages HTTP endpoint on the given port.
func Run(asyncErrorChannel chan<- error, port int) (closeFn func() error, err error) {
	zPagesMux := http.NewServeMux()
	zpages.Handle(zPagesMux, "/debug")

	addr := fmt.Sprintf(":%d", port)
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

	return srv.Close, nil
}
