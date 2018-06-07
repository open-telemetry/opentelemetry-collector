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

// Program opencensusd collects OpenCensus stats and traces
// to export to a configured backend.
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"

	pb "github.com/census-instrumentation/opencensus-proto/gen-go/exporterproto"
	"github.com/census-instrumentation/opencensus-service/internal"
	"google.golang.org/grpc"
)

func main() {
	ls, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		log.Fatalf("Cannot listen: %v", err)
	}

	endpointFile := internal.DefaultEndpointFile()
	if err := os.MkdirAll(filepath.Dir(endpointFile), 0755); err != nil {
		log.Fatalf("Cannot make directory for the endpoint file: %v", err)
	}
	if err := ioutil.WriteFile(endpointFile, []byte(ls.Addr().String()), 0777); err != nil {
		log.Fatalf("Cannot write the endpoint file: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		os.Remove(endpointFile)
		os.Exit(0)
	}()

	s := grpc.NewServer()
	pb.RegisterExportServer(s, &server{})
	if err := s.Serve(ls); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

type server struct{}

func (s *server) ExportSpan(stream pb.Export_ExportSpanServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// TODO(jbd): Implement.
		fmt.Println(in)
	}
}

func (s *server) ExportMetrics(stream pb.Export_ExportMetricsServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// TODO(jbd): Implement.
		fmt.Println(in)
	}
}

// TODO(jbd): Implement exporting to a backend.
