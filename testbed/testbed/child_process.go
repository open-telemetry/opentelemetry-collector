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

package testbed

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
)

// resourceSpec is a resource consumption specification.
type resourceSpec struct {
	// Percentage of one core the process is expected to consume at most.
	// Test is aborted and failed if consumption during
	// resourceCheckPeriod exceeds this number. If 0 the CPU
	// consumption is not monitored and does not affect the test result.
	expectedMaxCPU uint32

	// Maximum RAM in MiB the process is expected to consume.
	// Test is aborted and failed if consumption exceeds this number.
	// If 0 memory consumption is not monitored and does not affect
	// the test result.
	expectedMaxRAM uint32

	// Period during which CPU and RAM of the process are measured.
	// Bigger numbers will result in more averaging of short spikes.
	resourceCheckPeriod time.Duration
}

// isSpecified returns true if any part of resourceSpec is specified,
// i.e. has non-zero value.
func (rs *resourceSpec) isSpecified() bool {
	return rs != nil && (rs.expectedMaxCPU != 0 || rs.expectedMaxRAM != 0)
}

// childProcess is a child process that can be monitored and the output
// of which will be written to a log file.
type childProcess struct {
	// Descriptive name of the process
	name string

	// Command to execute
	cmd *exec.Cmd

	// WaitGroup for copying process output
	outputWG sync.WaitGroup

	// Various starting/stopping flags
	isStarted  bool
	stopOnce   sync.Once
	isStopped  bool
	doneSignal chan struct{}

	// Resource specification that must be monitored for.
	resourceSpec *resourceSpec

	// Process monitoring data.
	processMon *process.Process

	// Time when process was started.
	startTime time.Time

	// Last tick time we monitored the process.
	lastElapsedTime time.Time

	// Process times that were fetched on last monitoring tick.
	lastProcessTimes *cpu.TimesStat

	// Current RAM RSS in MiBs
	ramMiBCur uint32

	// Current CPU percentage times 1000 (we use scaling since we have to use int for atomic operations).
	cpuPercentX1000Cur uint32

	// Maximum CPU seen
	cpuPercentMax float64

	// Number of memory measurements
	memProbeCount int

	// Cumulative RAM RSS in MiBs
	ramMiBTotal uint64

	// Maximum RAM seen
	ramMiBMax uint32
}

type startParams struct {
	name         string
	logFilePath  string
	cmd          string
	cmdArgs      []string
	resourceSpec *resourceSpec
}

type ResourceConsumption struct {
	CPUPercentAvg float64
	CPUPercentMax float64
	RAMMiBAvg     uint32
	RAMMiBMax     uint32
}

// start a child process.
//
// Parameters:
// name is the human readable name of the process (e.g. "Agent"), used for logging.
// logFilePath is the file path to write the standard output and standard error of
// the process to.
// cmd is the executable to run.
// cmdArgs is the command line arguments to pass to the process.
func (cp *childProcess) start(params startParams) error {

	cp.name = params.name
	cp.doneSignal = make(chan struct{})
	cp.resourceSpec = params.resourceSpec

	log.Printf("Starting %s (%s)", cp.name, params.cmd)

	// Prepare log file
	logFile, err := os.Create(params.logFilePath)
	if err != nil {
		return fmt.Errorf("cannot create %s: %s", params.logFilePath, err.Error())
	}
	log.Printf("Writing %s log to %s", cp.name, params.logFilePath)

	// Prepare to start the process.
	cp.cmd = exec.Command(params.cmd, params.cmdArgs...)

	// Capture standard output and standard error.
	stdoutIn, err := cp.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cannot capture stdout of %s: %s", params.cmd, err.Error())
	}
	stderrIn, err := cp.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cannot capture stderr of %s: %s", params.cmd, err.Error())
	}

	// Start the process.
	if err := cp.cmd.Start(); err != nil {
		return fmt.Errorf("cannot start executable at %s: %s", params.cmd, err.Error())
	}

	cp.startTime = time.Now()
	cp.isStarted = true

	log.Printf("%s running, pid=%d", cp.name, cp.cmd.Process.Pid)

	// Create a WaitGroup that waits for both outputs to be finished copying.
	cp.outputWG.Add(2)

	// Begin copying outputs.
	go func() {
		_, _ = io.Copy(logFile, stdoutIn)
		cp.outputWG.Done()
	}()
	go func() {
		_, _ = io.Copy(logFile, stderrIn)
		cp.outputWG.Done()
	}()

	return nil
}

func (cp *childProcess) stop() {
	cp.stopOnce.Do(func() {

		if !cp.isStarted {
			// Process wasn't started, nothing to stop.
			return
		}

		cp.isStopped = true

		log.Printf("Gracefully terminating %s pid=%d, sending SIGTEM...", cp.name, cp.cmd.Process.Pid)

		// Notify resource monitor to stop.
		close(cp.doneSignal)

		// Gracefully signal process to stop.
		if err := cp.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("Cannot send SIGTEM: %s", err.Error())
		}

		finished := make(chan struct{})

		// Setup a goroutine to wait a while for process to finish and send kill signal
		// to the process if it doesn't finish.
		go func() {
			// Wait 10 seconds.
			t := time.After(10 * time.Second)
			select {
			case <-t:
				// Time is out. Kill the process.
				log.Printf("%s pid=%d is not responding to SIGTERM. Sending SIGKILL to kill forcedly.",
					cp.name, cp.cmd.Process.Pid)
				if err := cp.cmd.Process.Signal(syscall.SIGKILL); err != nil {
					log.Printf("Cannot send SIGKILL: %s", err.Error())
				}
			case <-finished:
				// Process is successfully finished.
			}
		}()

		// Wait for output to be fully copied.
		cp.outputWG.Wait()

		// Wait for process to terminate
		err := cp.cmd.Wait()

		// Let goroutine know process is finished.
		close(finished)

		// Set resource consumption stats to 0
		atomic.StoreUint32(&cp.ramMiBCur, 0)
		atomic.StoreUint32(&cp.cpuPercentX1000Cur, 0)

		log.Printf("%s process stopped, exit code=%d", cp.name, cp.cmd.ProcessState.ExitCode())

		if err != nil {
			log.Printf("%s execution failed: %s", cp.name, err.Error())
		}
	})
}

func (cp *childProcess) watchResourceConsumption() error {
	if !cp.resourceSpec.isSpecified() {
		// Resource monitoring is not enabled.
		return nil
	}

	var err error
	cp.processMon, err = process.NewProcess(int32(cp.cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("cannot monitor process %d: %s",
			cp.cmd.Process.Pid, err.Error())
	}

	cp.fetchRAMUsage()

	// Begin measuring elapsed and process CPU times.
	cp.lastElapsedTime = time.Now()
	cp.lastProcessTimes, err = cp.processMon.Times()
	if err != nil {
		return fmt.Errorf("cannot get process times for %d: %s",
			cp.cmd.Process.Pid, err.Error())
	}

	// Measure every resourceCheckPeriod.
	ticker := time.NewTicker(cp.resourceSpec.resourceCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.fetchRAMUsage()
			cp.fetchCPUUsage()

			if err := cp.checkAllowedResourceUsage(); err != nil {
				cp.stop()
				return err
			}

		case <-cp.doneSignal:
			log.Printf("Stopping process monitor.")
			return nil
		}
	}
}

func (cp *childProcess) fetchRAMUsage() {
	// Get process memory and CPU times
	mi, err := cp.processMon.MemoryInfo()
	if err != nil {
		log.Printf("cannot get process memory for %d: %s",
			cp.cmd.Process.Pid, err.Error())
		return
	}

	// Calculate RSS in MiBs.
	ramMiBCur := uint32(mi.RSS / mibibyte)

	// Calculate aggregates.
	cp.memProbeCount++
	cp.ramMiBTotal += uint64(ramMiBCur)
	if ramMiBCur > cp.ramMiBMax {
		cp.ramMiBMax = ramMiBCur
	}

	// Store current usage.
	atomic.StoreUint32(&cp.ramMiBCur, ramMiBCur)
}

func (cp *childProcess) fetchCPUUsage() {
	times, err := cp.processMon.Times()
	if err != nil {
		log.Printf("cannot get process times for %d: %s",
			cp.cmd.Process.Pid, err.Error())
		return
	}

	now := time.Now()

	// Calculate elapsed and process CPU time deltas in seconds
	deltaElapsedTime := now.Sub(cp.lastElapsedTime).Seconds()
	deltaCPUTime := times.Total() - cp.lastProcessTimes.Total()

	cp.lastProcessTimes = times
	cp.lastElapsedTime = now

	// Calculate CPU usage percentage in elapsed period.
	cpuPercent := deltaCPUTime * 100 / deltaElapsedTime
	if cpuPercent > cp.cpuPercentMax {
		cp.cpuPercentMax = cpuPercent
	}

	curCPUPercentageX1000 := uint32(cpuPercent * 1000)

	// Store current usage.
	atomic.StoreUint32(&cp.cpuPercentX1000Cur, curCPUPercentageX1000)
}

func (cp *childProcess) checkAllowedResourceUsage() error {
	// Check if current CPU usage exceeds expected.
	var errMsg string
	if cp.resourceSpec.expectedMaxCPU != 0 && cp.cpuPercentX1000Cur/1000 > cp.resourceSpec.expectedMaxCPU {
		errMsg = fmt.Sprintf("CPU consumption is %.1f%%, max expected is %d%%",
			float64(cp.cpuPercentX1000Cur)/1000.0, cp.resourceSpec.expectedMaxCPU)
	}

	// Check if current RAM usage exceeds expected.
	if cp.resourceSpec.expectedMaxRAM != 0 && cp.ramMiBCur > cp.resourceSpec.expectedMaxRAM {
		errMsg = fmt.Sprintf("RAM consumption is %d MiB, max expected is %d MiB",
			cp.ramMiBCur, cp.resourceSpec.expectedMaxRAM)
	}

	if errMsg == "" {
		return nil
	}

	log.Printf("Performance error: %s", errMsg)

	return errors.New(errMsg)
}

// GetResourceConsumption returns resource consumption as a string
func (cp *childProcess) GetResourceConsumption() string {
	if !cp.resourceSpec.isSpecified() {
		// Monitoring is not enabled.
		return ""
	}

	curRSSMib := atomic.LoadUint32(&cp.ramMiBCur)
	curCPUPercentageX1000 := atomic.LoadUint32(&cp.cpuPercentX1000Cur)

	return fmt.Sprintf("%s RAM (RES):%4d MiB, CPU:%4.1f%%", cp.name,
		curRSSMib, float64(curCPUPercentageX1000)/1000.0)
}

// GetTotalConsumption returns total resource consumption since start of process
func (cp *childProcess) GetTotalConsumption() *ResourceConsumption {
	rc := &ResourceConsumption{}

	if cp.processMon != nil {
		// Get total elapsed time since process start
		elapsedDuration := cp.lastElapsedTime.Sub(cp.startTime).Seconds()

		if elapsedDuration > 0 {
			// Calculate average CPU usage since start of process
			rc.CPUPercentAvg = cp.lastProcessTimes.Total() / elapsedDuration * 100.0
		}
		rc.CPUPercentMax = cp.cpuPercentMax

		if cp.memProbeCount > 0 {
			// Calculate average RAM usage by averaging all RAM measurements
			rc.RAMMiBAvg = uint32(cp.ramMiBTotal / uint64(cp.memProbeCount))
		}
		rc.RAMMiBMax = cp.ramMiBMax
	}

	return rc
}
