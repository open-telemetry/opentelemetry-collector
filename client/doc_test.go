// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"context"
	"fmt"
	"net"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func Example_receiver() {
	// Your receiver get a next consumer when it's constructed
	next, err := consumer.NewTraces(func(_ context.Context, _ ptrace.Traces) error {
		return nil
	})
	if err != nil {
		panic(err)
	}

	// You'll convert the incoming data into pipeline data
	td := ptrace.NewTraces()

	// You probably have a context with client metadata from your listener or
	// scraper
	ctx := context.Background()

	// Get the client from the context: if it doesn't exist, FromContext will
	// create one
	cl := client.FromContext(ctx)

	// Extract the client information based on your original context and set it
	// to Addr
	//nolint:govet
	cl.Addr = &net.IPAddr{
		IP: net.IPv4(1, 2, 3, 4),
	}

	// When you are done, propagate the context down the pipeline to the next
	// consumer and handle error.
	if err = next.ConsumeTraces(ctx, td); err != nil {
		panic(err)
	}
}

func Example_processor() {
	// Your processor or exporter will receive a context, from which you get the
	// client information
	ctx := context.Background()
	cl := client.FromContext(ctx)

	// And use the information from the client as you need
	fmt.Println(cl.Addr)
}

func Example_authenticator() {
	// Your configauth.AuthenticateFunc receives a context
	ctx := context.Background()

	// Get the client from the context: if it doesn't exist, FromContext will
	// create one
	cl := client.FromContext(ctx)

	// After a successful authentication, place the data you want to propagate
	// as part of an AuthData implementation of your own
	cl.Auth = &exampleAuthData{
		username: "jdoe",
	}

	// Your configauth.AuthenticateFunc should return this new context
	_ = client.NewContext(ctx, cl)
}

type exampleAuthData struct {
	username string
}

func (e *exampleAuthData) GetAttribute(key string) any {
	if key == "username" {
		return e.username
	}
	return nil
}

func (e *exampleAuthData) GetAttributeNames() []string {
	return []string{"username"}
}
