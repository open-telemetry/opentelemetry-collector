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

package pprofextension

import (
	"context"
	"errors"
	"net"
	"net/http"
	_ "net/http/pprof" // #nosec Needed to enable the performance profiler
	"os"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"unsafe"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

// Tracks that only a single instance is active per process.
// See comment on Start method for the reasons for that.
var activeInstance *pprofExtension

// #nosec G103
var activeInstancePtr = (*unsafe.Pointer)(unsafe.Pointer(&activeInstance))

type pprofExtension struct {
	config Config
	logger *zap.Logger
	file   *os.File
	server http.Server
	stopCh chan struct{}
}

func (p *pprofExtension) Start(_ context.Context, host component.Host) error {
	// The runtime settings are global to the application, so while in principle it
	// is possible to have more than one instance, running multiple will mean that
	// the settings of the last started instance will prevail. In order to avoid
	// this issue we will allow the start of a single instance once per process
	// Summary: only a single instance can be running in the same process.
	// #nosec G103
	if !atomic.CompareAndSwapPointer(activeInstancePtr, nil, unsafe.Pointer(p)) {
		return errors.New("only a single pprof extension instance can be running per process")
	}

	// Take care that if any error happen when starting the active instance is cleaned.
	var startErr error
	defer func() {
		if startErr != nil {
			atomic.StorePointer(activeInstancePtr, nil)
		}
	}()

	// Start the listener here so we can have earlier failure if port is
	// already in use.
	var ln net.Listener
	ln, startErr = p.config.TCPAddr.Listen()
	if startErr != nil {
		return startErr
	}

	runtime.SetBlockProfileRate(p.config.BlockProfileFraction)
	runtime.SetMutexProfileFraction(p.config.MutexProfileFraction)

	p.logger.Info("Starting net/http/pprof server", zap.Any("config", p.config))
	p.stopCh = make(chan struct{})
	go func() {
		defer close(p.stopCh)

		// The listener ownership goes to the server.
		err := p.server.Serve(ln)
		atomic.StorePointer(activeInstancePtr, nil)
		if err != nil && err != http.ErrServerClosed {
			host.ReportFatalError(err)
		}
	}()

	if p.config.SaveToFile != "" {
		var f *os.File
		f, startErr = os.Create(p.config.SaveToFile)
		if startErr != nil {
			return startErr
		}
		p.file = f
		startErr = pprof.StartCPUProfile(f)
	}

	return startErr
}

func (p *pprofExtension) Shutdown(context.Context) error {
	defer atomic.StorePointer(activeInstancePtr, nil)
	if p.file != nil {
		pprof.StopCPUProfile()
		_ = p.file.Close() // ignore the error
	}
	err := p.server.Close()
	if p.stopCh != nil {
		<-p.stopCh
	}
	return err
}

func newServer(config Config, logger *zap.Logger) *pprofExtension {
	return &pprofExtension{
		config: config,
		logger: logger,
	}
}
