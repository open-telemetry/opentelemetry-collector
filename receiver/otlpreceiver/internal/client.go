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

package internal // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal"

import (
	"context"

	"go.opentelemetry.io/collector/client"
	"google.golang.org/grpc/peer"
)

// ContextWithClient returns a context with either a new or an existing client.Client,
// enhanced with the client IP address extracted from the provided context.
func ContextWithClient(ctx context.Context) context.Context {
	cl := client.FromContext(ctx)
	if p, ok := peer.FromContext(ctx); ok {
		ip := ParseIP(p.Addr.String())
		if ip != "" {
			cl.IP = ip
		}
	}
	return client.NewContext(ctx, cl)
}
