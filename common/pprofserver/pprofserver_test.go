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

package pprofserver

import (
	"flag"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func TestPerformanceProfilerFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	AddFlags(fs)

	args := []string{
		"--" + httpPprofPortCfg + "=1777",
		"--" + pprofBlockProfileFraction + "=5",
		"--" + pprofMutexProfileFraction + "=-1",
	}

	if err := fs.Parse(args); err != nil {
		t.Fatalf("failed to parse arguments: %v", err)
	}
}

func TestPerformanceProfilerServer(t *testing.T) {
	v := viper.New()
	const pprofPort = 17788
	v.Set(httpPprofPortCfg, pprofPort)
	v.Set(pprofBlockProfileFraction, 3)
	v.Set(pprofMutexProfileFraction, 5)

	asyncErrChan := make(chan error, 1)
	if err := SetupFromViper(asyncErrChan, v, zap.NewNop()); err != nil {
		t.Fatalf("failed to setup pprof server: %v", err)
	}

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	client := &http.Client{}
	resp, err := client.Get("http://localhost:" + strconv.Itoa(pprofPort) + "/debug/pprof")
	if err != nil {
		t.Fatalf("failed to get a response from pprof server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ppropf server response: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	select {
	case err := <-asyncErrChan:
		t.Fatalf("async err received from pprof: %v", err)
	case <-time.After(250 * time.Millisecond):
	}
}
