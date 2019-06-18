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

package zpagesserver

import (
	"flag"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestZPagesServerFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	AddFlags(fs)

	args := []string{
		"--" + ZPagesHTTPPort + "=12345",
	}

	if err := fs.Parse(args); err != nil {
		t.Fatalf("failed to parse arguments: %v", err)
	}
}

func TestZPagesServerPortInUse(t *testing.T) {
	const zpagesPort = 17789
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(zpagesPort))
	if err != nil {
		t.Fatalf("error opening port: %v", err)
	}
	defer ln.Close()
	asyncErrChan := make(chan error)
	closeFn, err := Run(asyncErrChan, zpagesPort)
	if err == nil {
		closeFn()
		t.Fatalf("expected error, got nil")
	}
}

func TestZPagesServer(t *testing.T) {
	const zpagesPort = 17789

	asyncErrChan := make(chan error, 1)
	closeFn, err := Run(asyncErrChan, zpagesPort)
	if err != nil {
		t.Fatalf("failed to setup zpages server: %v", err)
	}
	defer closeFn()

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	client := &http.Client{}
	resp, err := client.Get("http://localhost:" + strconv.Itoa(zpagesPort) + "/debug/tracez")
	if err != nil {
		t.Fatalf("failed to get a response from zpages server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("zpages server response: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	select {
	case err := <-asyncErrChan:
		t.Fatalf("async err received from zpages: %v", err)
	case <-time.After(250 * time.Millisecond):
	}
}
