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

// Package client contains generic representations of clients connecting to
// different receivers. Components, such as processors or exporters, can make
// use of this information to make decisions related to grouping of batches,
// tenancy, load balancing, tagging, among others.
//
// The structs defined here are typically used within the context that is
// propagated down the pipeline, with the values being produced by receivers,
// and consumed by processors and exporters.
//
// Producers
//
// Receivers are responsible for obtaining a client.Info from the current
// context and enhancing the client.Info with the net.Addr from the peer,
// storing a new client.Info into the context that it passes down. For HTTP
// requests, the net.Addr is typically the IP address of the client.
//
// Typically, however, receivers would delegate this processing to helpers such
// as the confighttp or configgrpc packages: both contain interceptors that will
// enhance the context with the client.Info, such that no actions are needed by
// receivers that are built using confighttp.HTTPServerSettings or
// configgrpc.GRPCServerSettings.
//
// Consumers
//
// Provided that the pipeline does not contain processors that would discard or
// rewrite the context, such as the batch processor, processors and exporters
// have access to the client.Info via client.FromContext. Among other usages,
// this data can be used to:
//
// - annotate data points with authentication data (username, tenant, ...)
//
// - route data points based on authentication data
//
// - rate limit client calls based on IP addresses
//
// Processors and exporters relying on the existence of data from the
// client.Info, should clearly document this as part of the component's README
// file.
package client // import "go.opentelemetry.io/collector/client"

import (
	"context"
	"net"
	"net/http"

	"google.golang.org/grpc/peer"
)

type ctxKey struct{}

// Info contains data related to the clients connecting to receivers.
type Info struct {
	// Addr for the client connecting to this collector. Available in a
	// best-effort basis, and generally reliable for receivers making use of
	// confighttp.ToServer and configgrpc.ToServerOption.
	Addr net.Addr
}

// NewContext takes an existing context and derives a new context with the
// client.Info value stored on it.
func NewContext(ctx context.Context, c Info) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// FromContext takes a context and returns a ClientInfo from it.
// When a ClientInfo isn't present, a new empty one is returned.
func FromContext(ctx context.Context) Info {
	c, ok := ctx.Value(ctxKey{}).(Info)
	if !ok {
		c = Info{}
	}
	return c
}

// FromGRPC takes a GRPC context and tries to extract client information from it
func FromGRPC(ctx context.Context) (Info, bool) {
	if p, ok := peer.FromContext(ctx); ok {
		ip := parseIP(p.Addr.String())
		if ip != nil {
			return Info{ip}, true
		}
	}
	return Info{}, false
}

// FromHTTP takes a net/http Request object and tries to extract client information from it
func FromHTTP(r *http.Request) (Info, bool) {
	ip := parseIP(r.RemoteAddr)
	if ip == nil {
		return Info{}, false
	}
	return Info{ip}, true
}

func parseIP(source string) net.Addr {
	ipstr, _, err := net.SplitHostPort(source)
	if err == nil {
		source = ipstr
	}
	return &net.IPAddr{
		IP: net.ParseIP(source),
	}
}
