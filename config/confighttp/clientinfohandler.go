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

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"net/http"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/config/internal"
)

var _ http.Handler = (*clientInfoHandler)(nil)

type clientInfoHandler struct {
	next http.Handler
}

func (h *clientInfoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(contextWithClient(req))
	h.next.ServeHTTP(w, req)
}

func contextWithClient(req *http.Request) context.Context {
	cl := client.FromContext(req.Context())

	ip := internal.ParseIP(req.RemoteAddr)
	if ip != nil {
		cl.IP = ip
	}

	ctx := client.NewContext(req.Context(), cl)
	return ctx
}
