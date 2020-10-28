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

package fluentbitextension

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const mockScript = `#!/bin/sh

echo "Config:" 1>&2
cat -

sleep 100

`

func setup(t *testing.T, conf *Config) (*processManager, **process.Process, func() bool, func()) {
	logCore, logObserver := observer.New(zap.DebugLevel)
	logger := zap.New(logCore)

	mockScriptFile, err := ioutil.TempFile("", "mocksubproc")
	require.Nil(t, err)

	cleanup := func() {
		spew.Dump(logObserver.All())
		os.Remove(mockScriptFile.Name())
	}

	_, err = mockScriptFile.Write([]byte(mockScript))
	require.Nil(t, err)

	err = mockScriptFile.Chmod(0700)
	require.Nil(t, err)

	require.NoError(t, mockScriptFile.Close())

	conf.ExecutablePath = mockScriptFile.Name()
	pm := newProcessManager(conf, logger)

	var mockProc *process.Process
	findSubproc := func() bool {
		selfPid := os.Getpid()
		procs, _ := process.Processes()
		for _, proc := range procs {
			if ppid, _ := proc.Ppid(); ppid == int32(selfPid) {
				cmdline, _ := proc.Cmdline()
				if strings.HasPrefix(cmdline, "/bin/sh "+mockScriptFile.Name()) {
					mockProc = proc
					return true
				}
			}
		}
		return false
	}

	return pm, &mockProc, findSubproc, cleanup
}

func TestProcessManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pm, mockProc, findSubproc, cleanup := setup(t, &Config{
		TCPEndpoint: "127.0.0.1:8000",
		Config:      "example config",
	})
	defer cleanup()

	pm.Start(ctx, nil)
	defer pm.Shutdown(ctx)

	require.Eventually(t, findSubproc, 12*time.Second, 100*time.Millisecond)
	require.NotNil(t, *mockProc)

	cmdline, err := (*mockProc).Cmdline()
	require.Nil(t, err)
	require.Equal(t,
		"/bin/sh "+pm.conf.ExecutablePath+
			" --config=/dev/stdin --http --port=2020 --flush=1 -o forward://127.0.0.1:8000 --match=*",
		cmdline)

	oldProcPid := (*mockProc).Pid
	err = (*mockProc).Kill()
	require.NoError(t, err)

	// Should be restarted
	require.Eventually(t, findSubproc, restartDelay+3*time.Second, 100*time.Millisecond)
	require.NotNil(t, *mockProc)

	require.NotEqual(t, (*mockProc).Pid, oldProcPid)
}

func TestProcessManagerArgs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pm, mockProc, findSubproc, cleanup := setup(t, &Config{
		TCPEndpoint: "127.0.0.1:8000",
		Config:      "example config",
		Args:        []string{"--http"},
	})
	defer cleanup()

	pm.Start(ctx, nil)
	defer pm.Shutdown(ctx)

	require.Eventually(t, findSubproc, 12*time.Second, 100*time.Millisecond)
	require.NotNil(t, *mockProc)

	cmdline, err := (*mockProc).Cmdline()
	require.Nil(t, err)
	require.Equal(t,
		"/bin/sh "+pm.conf.ExecutablePath+
			" --http",
		cmdline)
}

func TestProcessManagerBadExec(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logCore, logObserver := observer.New(zap.DebugLevel)
	logger := zap.New(logCore)

	pm := newProcessManager(&Config{
		ExecutablePath: "/does/not/exist",
		TCPEndpoint:    "127.0.0.1:8000",
		Config:         "example config",
	}, logger)

	pm.Start(ctx, nil)
	defer pm.Shutdown(ctx)

	time.Sleep(restartDelay + 2*time.Second)
	require.Len(t, logObserver.FilterMessage("FluentBit process died").All(), 2)
}

func TestProcessManagerEmptyConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pm, mockProc, findSubproc, cleanup := setup(t, &Config{
		TCPEndpoint: "127.0.0.1:8000",
		Config:      "",
	})
	defer cleanup()

	pm.Start(ctx, nil)
	defer pm.Shutdown(ctx)

	require.Eventually(t, findSubproc, 15*time.Second, 100*time.Millisecond)
	require.NotNil(t, *mockProc)
}
