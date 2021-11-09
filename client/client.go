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

// Package client contains generic representations of clients connecting to different receivers. Components,
// such as processors or exporters, can make use of this information to make decisions related to
// grouping of batches, tenancy, load balancing, tagging, among others.
package client // import "go.opentelemetry.io/collector/client"

import (
	"context"
	"net"
)

type ctxKey struct{}

// Info contains data related to the clients connecting to receivers.
type Info struct {
	// Addr for the client connecting to this collector. Available in a best-effort basis, and generally reliable
	// for receivers making use of confighttp.ToServer and configgrpc.ToServerOption.
	Addr net.Addr

	// Auth information from the incoming request as provided by configauth.ServerAuthenticator implementations
	// tied to the receiver for this connection.
	Auth AuthData
}

// AuthData represents the authentication data as seen by authenticators tied to the receivers.
type AuthData interface {
	// GetAttribute returns the value for the given attribute.
	GetAttribute(string) interface{}

	// GetAttributes returns the names of all attributes in this authentication data.
	GetAttributeNames() []string
}

// NewContext takes an existing context and derives a new context with the client.Info value stored on it.
func NewContext(ctx context.Context, c *Info) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// FromContext takes a context and returns a ClientInfo from it. When a ClientInfo isn't present, a new
// empty one is returned.
func FromContext(ctx context.Context) *Info {
	c, ok := ctx.Value(ctxKey{}).(*Info)
	if !ok {
		c = &Info{}
	}
	return c
}
