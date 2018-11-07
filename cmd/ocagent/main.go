// Copyright 2018, OpenCensus Authors
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

// Program ocagent collects OpenCensus stats and traces
// to export to a configured backend.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/interceptor/octrace"
	"github.com/census-instrumentation/opencensus-service/interceptor/zipkin"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/spanreceiver"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
)

var configYAMLFile string
var ocInterceptorPort int

const zipkinRoute = "/api/v2/spans"

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runOCAgent() {
	yamlBlob, err := ioutil.ReadFile(configYAMLFile)
	if err != nil {
		log.Fatalf("Cannot read the YAML file %v error: %v", configYAMLFile, err)
	}
	agentConfig, err := parseOCAgentConfig(yamlBlob)
	if err != nil {
		log.Fatalf("Failed to parse own configuration %v error: %v", configYAMLFile, err)
	}

	// Ensure that we check and catch any logical errors with the
	// configuration e.g. if an interceptor shares the same address
	// as an exporter which would cause a self DOS and waste resources.
	if err := agentConfig.checkLogicalConflicts(yamlBlob); err != nil {
		log.Fatalf("Configuration logical error: %v", err)
	}

	ocInterceptorAddr := agentConfig.ocInterceptorAddress()

	traceExporters, closeFns := exportersFromYAMLConfig(yamlBlob)
	commonSpanReceiver := exporter.MultiTraceExporters(traceExporters...)

	// Add other interceptors here as they are implemented
	ocInterceptorDoneFn, err := runOCInterceptor(ocInterceptorAddr, commonSpanReceiver)
	if err != nil {
		log.Fatal(err)
	}
	closeFns = append(closeFns, ocInterceptorDoneFn)

	// If zPages are enabled, run them
	zPagesPort, zPagesEnabled := agentConfig.zPagesPort()
	if zPagesEnabled {
		zCloseFn := runZPages(zPagesPort)
		closeFns = append(closeFns, zCloseFn)
	}

	// If the Zipkin interceptor is enabled, then run it
	if agentConfig.zipkinInterceptorEnabled() {
		zipkinInterceptorAddr := agentConfig.zipkinInterceptorAddress()
		zipkinInterceptorDoneFn, err := runZipkinInterceptor(zipkinInterceptorAddr, commonSpanReceiver)
		if err != nil {
			log.Fatal(err)
		}
		closeFns = append(closeFns, zipkinInterceptorDoneFn)
	}

	// Always cleanup finally
	defer func() {
		for _, closeFn := range closeFns {
			if closeFn != nil {
				closeFn()
			}
		}
	}()

	signalsChan := make(chan os.Signal)
	signal.Notify(signalsChan, os.Interrupt)

	// Wait for the closing signal
	<-signalsChan
}

func runZPages(port int) func() error {
	// And enable zPages too
	zPagesMux := http.NewServeMux()
	zpages.Handle(zPagesMux, "/debug")

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to bind to run zPages on %q: %v", addr, err)
	}

	srv := http.Server{Handler: zPagesMux}
	go func() {
		log.Printf("Running zPages at %q", addr)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve zPages: %v", err)
		}
	}()

	return srv.Close
}

func runOCInterceptor(addr string, sr spanreceiver.SpanReceiver) (doneFn func() error, err error) {
	oci, err := octrace.New(sr, octrace.WithSpanBufferPeriod(800*time.Millisecond))
	if err != nil {
		return nil, fmt.Errorf("Failed to create the OpenCensus interceptor: %v", err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Cannot bind to address %q: %v", addr, err)
	}
	srv := internal.GRPCServerWithObservabilityEnabled()
	if err := view.Register(internal.AllViews...); err != nil {
		return nil, fmt.Errorf("Failed to register internal.AllViews: %v", err)
	}
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		return nil, fmt.Errorf("Failed to register ocgrpc.DefaultServerViews: %v", err)
	}

	agenttracepb.RegisterTraceServiceServer(srv, oci)
	go func() {
		log.Printf("Running OpenCensus interceptor as a gRPC service at %q", addr)
		if err := srv.Serve(ln); err != nil {
			log.Fatalf("Failed to run OpenCensus interceptor: %v", err)
		}
	}()
	doneFn = func() error {
		srv.Stop()
		return nil
	}
	return doneFn, nil
}

func runZipkinInterceptor(addr string, sr spanreceiver.SpanReceiver) (doneFn func() error, err error) {
	zi, err := zipkininterceptor.New(sr)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the Zipkin interceptor: %v", err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Cannot bind Zipkin interceptor to address %q: %v", addr, err)
	}
	mux := http.NewServeMux()
	mux.Handle(zipkinRoute, zi)
	go func() {
		fullAddr := addr + zipkinRoute
		log.Printf("Running the Zipkin interceptor at %q", fullAddr)
		if err := http.Serve(ln, mux); err != nil {
			log.Fatalf("Failed to serve the Zipkin interceptor: %v", err)
		}
	}()

	doneFn = ln.Close
	return doneFn, nil
}
