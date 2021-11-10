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

package client_test

import (
	"context"
	"fmt"
	"net"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

func Example_receiver() {
	// Your receiver get a next consumer when it's constructed
	var next consumer.Traces

	// You'll convert the incoming data into pipeline data
	td := pdata.NewTraces()

	// You probably have a context with client metadata from your listener or scraper
	ctx := context.Background()

	// Get the client from the context: if it doesn't exist, FromContext will create one
	cl := client.FromContext(ctx)

	// Extract the client information based on your original context and set it to Addr
	cl.Addr = &net.IPAddr{
		IP: net.IPv4(1, 2, 3, 4),
	}

	// When you are done, propagate the context down the pipeline to the next consumer
	next.ConsumeTraces(ctx, td)
}

func Example_authenticator() (context.Context, error) {
	// Your configauth.AuthenticateFunc receives a context
	ctx := context.Background()

	// Get the client from the context: if it doesn't exist, FromContext will create one
	cl := client.FromContext(ctx)

	// After a successful authentication, place the data you want to propagate as part of
	// an AuthData implementation of your own
	cl.Auth = &exampleAuthData{
		username: "jdoe",
	}

	// Your configauth.AuthenticateFunc should return this new context
	return client.NewContext(ctx, cl), nil
}

func Example_processor() {
	// Your processor or exporter will receive a context, from which you get the client information
	ctx := context.Background()
	cl := client.FromContext(ctx)

	// And use the information from the client as you need
	fmt.Println(cl.Addr)

	// Your component's configuration might have an option for your users to specify
	// the auth attribute name to use:
	value := cl.Auth.GetAttribute("username")
	fmt.Printf("Value: %v", value)

	// Or perhaps you want to print them all?
	for _, attr := range cl.Auth.GetAttributeNames() {
		value = cl.Auth.GetAttribute(attr)
		fmt.Printf("Key: %q, value: %v", attr, value)
	}
}

type exampleAuthData struct {
	username string
}

func (e *exampleAuthData) GetAttribute(string) interface{} {
	return nil
}
func (e *exampleAuthData) GetAttributeNames() []string {
	return nil
}
