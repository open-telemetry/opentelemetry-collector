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

package ocagent

import (
	"os"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.opencensus.io"

	"github.com/open-telemetry/opentelemetry-service/internal/version"
)

// NodeWithStartTime creates a node using nodeName and derives:
//  Hostname from the environment
//  Pid from the current process
//  StartTimestamp from the start time of this process
//  Language and library information.
func NodeWithStartTime(nodeName string) *commonpb.Node {
	return &commonpb.Node{
		Identifier: &commonpb.ProcessIdentifier{
			HostName:       os.Getenv("HOSTNAME"),
			Pid:            uint32(os.Getpid()),
			StartTimestamp: timeToTimestamp(startTime),
		},
		LibraryInfo: &commonpb.LibraryInfo{
			Language:           commonpb.LibraryInfo_GO_LANG,
			ExporterVersion:    version.Version,
			CoreLibraryVersion: opencensus.Version(),
		},
		ServiceInfo: &commonpb.ServiceInfo{
			Name: nodeName,
		},
		Attributes: make(map[string]string),
	}
}

func timeToTimestamp(t time.Time) *timestamp.Timestamp {
	nanoTime := t.UnixNano()
	return &timestamp.Timestamp{
		Seconds: nanoTime / 1e9,
		Nanos:   int32(nanoTime % 1e9),
	}
}
