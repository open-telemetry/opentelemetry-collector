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
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

type processManager struct {
	cancel         context.CancelFunc
	conf           *Config
	logger         *zap.Logger
	shutdownSignal chan struct{}
}

func newProcessManager(conf *Config, logger *zap.Logger) *processManager {
	return &processManager{
		conf:           conf,
		logger:         logger,
		shutdownSignal: make(chan struct{}),
	}
}

type procState string

// A global var that is available only for testing
var restartDelay = 10 * time.Second

const (
	Starting     procState = "starting"
	Running      procState = "running"
	ShuttingDown procState = "shutting-down"
	Stopped      procState = "stopped"
	Restarting   procState = "restarting"
	Errored      procState = "errored"
)

func constructArgs(tcpEndpoint string) []string {
	return []string{
		"--config=/dev/stdin",
		"--http",
		"--port=2020",
		"--flush=1",
		"-o", "forward://" + tcpEndpoint,
		"--match=*",
	}
}

func (pm *processManager) Start(ctx context.Context, _ component.Host) error {
	childCtx, cancel := context.WithCancel(ctx)
	pm.cancel = cancel

	args := pm.conf.Args
	if len(args) == 0 {
		args = constructArgs(pm.conf.TCPEndpoint)
	}
	go func() {
		run(childCtx, pm.conf.ExecutablePath, args, pm.conf.Config, pm.logger)
		close(pm.shutdownSignal)
	}()
	return nil
}

// Shutdown is invoked during service shutdown.
func (pm *processManager) Shutdown(context.Context) error {
	pm.cancel()
	t := time.NewTimer(5 * time.Second)

	// Wait for either the FluentBit process to terminate or the timeout
	// period, whichever comes first.
	select {
	case <-pm.shutdownSignal:
	case <-t.C:
	}

	return nil
}

func run(ctx context.Context, execPath string, args []string, config string, logger *zap.Logger) {
	state := Starting

	var cmd *exec.Cmd
	var err error
	var stdin io.WriteCloser
	var stdout io.ReadCloser
	// procWait is guaranteed to be sent exactly one message per successful process start
	procWait := make(chan error)

	// A state machine makes the management easier to understand and account
	// for all of the edge cases when managing a subprocess.
	for {
		logger.Debug("Fluent extension changed state", zap.String("state", string(state)))

		switch state {
		case Errored:
			logger.Error("FluentBit process died", zap.Error(err))
			state = Restarting

		case Starting:
			cmd, stdin, stdout = createCommand(execPath, args)

			logger.Debug("Starting fluent subprocess", zap.String("command", cmd.String()))
			err = cmd.Start()
			if err != nil {
				state = Errored
				continue
			}

			go signalWhenProcessDone(cmd, procWait)

			state = Running

		case Running:
			go collectOutput(stdout, logger)

			err = renderConfig(config, stdin)
			stdin.Close()
			if err != nil {
				state = Errored
				continue
			}

			select {
			case err = <-procWait:
				if ctx.Err() == nil {
					// We aren't supposed to shutdown yet so this is an error
					// state.
					state = Errored
					continue
				}
				state = Stopped
			case <-ctx.Done():
				state = ShuttingDown
			}

		case ShuttingDown:
			_ = cmd.Process.Signal(syscall.SIGTERM)
			<-procWait
			stdout.Close()
			state = Stopped

		case Restarting:
			_ = stdout.Close()
			_ = stdin.Close()

			// Sleep for a bit so we don't have a hot loop on repeated failures.
			time.Sleep(restartDelay)
			state = Starting

		case Stopped:
			return
		}
	}
}

func signalWhenProcessDone(cmd *exec.Cmd, procWait chan<- error) {
	err := cmd.Wait()
	procWait <- err
}

func renderConfig(config string, writer io.Writer) error {
	if config == "" {
		return nil
	}

	_, err := writer.Write([]byte(config))
	return err
}

func createCommand(execPath string, args []string) (*exec.Cmd, io.WriteCloser, io.ReadCloser) {
	cmd := exec.Command(execPath, args...)

	inReader, inWriter, err := os.Pipe()
	if err != nil {
		panic("Input pipe could not be created for subprocess")
	}

	cmd.Stdin = inReader

	outReader, outWriter, err := os.Pipe()
	// If this errors things are really wrong with the system
	if err != nil {
		panic("Output pipe could not be created for subprocess")
	}
	cmd.Stdout = outWriter
	cmd.Stderr = outWriter

	cmd.Env = os.Environ()

	applyOSSpecificCmdModifications(cmd)

	return cmd, inWriter, outReader
}

func collectOutput(stdout io.Reader, logger *zap.Logger) {
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		logger.Debug(scanner.Text())
	}
	// Returns when stdout is closed when the process ends
}
