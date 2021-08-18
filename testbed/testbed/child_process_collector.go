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

package testbed

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
	"go.uber.org/atomic"
)

// childProcessCollector implements the OtelcolRunner interface as a child process on the same machine executing
// the test. The process can be monitored and the output of which will be written to a log file.
type childProcessCollector struct {
	// Path to agent executable. If unset the default executable in
	// bin/otelcol_{{.GOOS}}_{{.GOARCH}} will be used.
	// Can be set for example to use the unstable executable for a specific test.
	AgentExePath string

	// Descriptive name of the process
	name string

	// Config file name
	configFileName string

	// Command to execute
	cmd *exec.Cmd

	// Various starting/stopping flags
	isStarted  bool
	stopOnce   sync.Once
	isStopped  bool
	doneSignal chan struct{}

	// Resource specification that must be monitored for.
	resourceSpec *ResourceSpec

	// Process monitoring data.
	processMon *process.Process

	// Time when process was started.
	startTime time.Time

	// Last tick time we monitored the process.
	lastElapsedTime time.Time

	// Process times that were fetched on last monitoring tick.
	lastProcessTimes *cpu.TimesStat

	// Current RAM RSS in MiBs
	ramMiBCur atomic.Uint32

	// Current CPU percentage times 1000 (we use scaling since we have to use int for atomic operations).
	cpuPercentX1000Cur atomic.Uint32

	// Maximum CPU seen
	cpuPercentMax float64

	// Number of memory measurements
	memProbeCount int

	// Cumulative RAM RSS in MiBs
	ramMiBTotal uint64

	// Maximum RAM seen
	ramMiBMax uint32
}

// NewChildProcessCollector crewtes a new OtelcolRunner as a child process on the same machine executing the test.
func NewChildProcessCollector() OtelcolRunner {
	return &childProcessCollector{}
}

func (cp *childProcessCollector) PrepareConfig(configStr string) (configCleanup func(), err error) {
	configCleanup = func() {
		// NoOp
	}
	var file *os.File
	file, err = ioutil.TempFile("", "agent*.yaml")
	if err != nil {
		log.Printf("%s", err)
		return configCleanup, err
	}

	defer func() {
		errClose := file.Close()
		if errClose != nil {
			log.Printf("%s", errClose)
		}
	}()

	if _, err = file.WriteString(configStr); err != nil {
		log.Printf("%s", err)
		return configCleanup, err
	}
	cp.configFileName = file.Name()
	configCleanup = func() {
		os.Remove(cp.configFileName)
	}
	return configCleanup, err
}

func expandExeFileName(exeName string) string {
	cfgTemplate, err := template.New("").Parse(exeName)
	if err != nil {
		log.Fatalf("Template failed to parse exe name %q: %s",
			exeName, err.Error())
	}

	templateVars := struct {
		GOOS   string
		GOARCH string
	}{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
	var buf bytes.Buffer
	if err = cfgTemplate.Execute(&buf, templateVars); err != nil {
		log.Fatalf("Configuration template failed to run on exe name %q: %s",
			exeName, err.Error())
	}

	return buf.String()
}

// Start a child process.
//
// cp.AgentExePath defines the executable to run. If unspecified
// "../../bin/otelcol_{{.GOOS}}_{{.GOARCH}}" will be used.
// {{.GOOS}} and {{.GOARCH}} will be expanded to the current OS and ARCH correspondingly.
//
// Parameters:
// name is the human readable name of the process (e.g. "Agent"), used for logging.
// logFilePath is the file path to write the standard output and standard error of
// the process to.
// cmdArgs is the command line arguments to pass to the process.
func (cp *childProcessCollector) Start(params StartParams) error {

	cp.name = params.Name
	cp.doneSignal = make(chan struct{})
	cp.resourceSpec = params.resourceSpec

	if cp.AgentExePath == "" {
		cp.AgentExePath = GlobalConfig.DefaultAgentExeRelativeFile
	}
	exePath := expandExeFileName(cp.AgentExePath)
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}

	log.Printf("Starting %s (%s)", cp.name, exePath)

	// Prepare log file
	logFile, err := os.Create(params.LogFilePath)
	if err != nil {
		return fmt.Errorf("cannot create %s: %s", params.LogFilePath, err.Error())
	}
	log.Printf("Writing %s log to %s", cp.name, params.LogFilePath)

	// Prepare to start the process.
	// #nosec
	args := params.CmdArgs
	if !containsConfig(args) {
		if cp.configFileName == "" {
			configFile := path.Join("testdata", "agent-config.yaml")
			cp.configFileName, err = filepath.Abs(configFile)
			if err != nil {
				return err
			}
		}
		args = append(args, "--config")
		args = append(args, cp.configFileName)
	}
	// #nosec
	cp.cmd = exec.Command(exePath, args...)

	// Capture standard output and standard error.
	cp.cmd.Stdout = logFile
	cp.cmd.Stderr = logFile

	// Start the process.
	if err = cp.cmd.Start(); err != nil {
		return fmt.Errorf("cannot start executable at %s: %s", exePath, err.Error())
	}

	cp.startTime = time.Now()
	cp.isStarted = true

	log.Printf("%s running, pid=%d", cp.name, cp.cmd.Process.Pid)

	return err
}

func (cp *childProcessCollector) Stop() (stopped bool, err error) {
	if !cp.isStarted || cp.isStopped {
		return false, nil
	}
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
		if err = cp.cmd.Process.Signal(syscall.SIGTERM); err != nil {
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
				if err = cp.cmd.Process.Signal(syscall.SIGKILL); err != nil {
					log.Printf("Cannot send SIGKILL: %s", err.Error())
				}
			case <-finished:
				// Process is successfully finished.
			}
		}()

		// Wait for process to terminate
		err = cp.cmd.Wait()

		// Let goroutine know process is finished.
		close(finished)

		// Set resource consumption stats to 0
		cp.ramMiBCur.Store(0)
		cp.cpuPercentX1000Cur.Store(0)

		log.Printf("%s process stopped, exit code=%d", cp.name, cp.cmd.ProcessState.ExitCode())

		if err != nil {
			log.Printf("%s execution failed: %s", cp.name, err.Error())
		}
	})
	stopped = true
	return stopped, err
}

func (cp *childProcessCollector) WatchResourceConsumption() error {
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

	// Measure every ResourceCheckPeriod.
	ticker := time.NewTicker(cp.resourceSpec.ResourceCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.fetchRAMUsage()
			cp.fetchCPUUsage()

			if err := cp.checkAllowedResourceUsage(); err != nil {
				if _, errStop := cp.Stop(); errStop != nil {
					log.Printf("Failed to stop child process: %v", err)
				}
				return err
			}

		case <-cp.doneSignal:
			log.Printf("Stopping process monitor.")
			return nil
		}
	}
}

func (cp *childProcessCollector) GetProcessMon() *process.Process {
	return cp.processMon
}

func (cp *childProcessCollector) fetchRAMUsage() {
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
	cp.ramMiBCur.Store(ramMiBCur)
}

func (cp *childProcessCollector) fetchCPUUsage() {
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
	if deltaCPUTime < 0 {
		// We sometimes get negative difference when the process is terminated.
		deltaCPUTime = 0
	}

	cp.lastProcessTimes = times
	cp.lastElapsedTime = now

	// Calculate CPU usage percentage in elapsed period.
	cpuPercent := deltaCPUTime * 100 / deltaElapsedTime
	if cpuPercent > cp.cpuPercentMax {
		cp.cpuPercentMax = cpuPercent
	}

	curCPUPercentageX1000 := uint32(cpuPercent * 1000)

	// Store current usage.
	cp.cpuPercentX1000Cur.Store(curCPUPercentageX1000)
}

func (cp *childProcessCollector) checkAllowedResourceUsage() error {
	// Check if current CPU usage exceeds expected.
	var errMsg string
	if cp.resourceSpec.ExpectedMaxCPU != 0 && cp.cpuPercentX1000Cur.Load()/1000 > cp.resourceSpec.ExpectedMaxCPU {
		errMsg = fmt.Sprintf("CPU consumption is %.1f%%, max expected is %d%%",
			float64(cp.cpuPercentX1000Cur.Load())/1000.0, cp.resourceSpec.ExpectedMaxCPU)
	}

	// Check if current RAM usage exceeds expected.
	if cp.resourceSpec.ExpectedMaxRAM != 0 && cp.ramMiBCur.Load() > cp.resourceSpec.ExpectedMaxRAM {
		errMsg = fmt.Sprintf("RAM consumption is %s MiB, max expected is %d MiB",
			cp.ramMiBCur.String(), cp.resourceSpec.ExpectedMaxRAM)
	}

	if errMsg == "" {
		return nil
	}

	log.Printf("Performance error: %s", errMsg)

	return errors.New(errMsg)
}

// GetResourceConsumption returns resource consumption as a string
func (cp *childProcessCollector) GetResourceConsumption() string {
	if !cp.resourceSpec.isSpecified() {
		// Monitoring is not enabled.
		return ""
	}

	curRSSMib := cp.ramMiBCur.Load()
	curCPUPercentageX1000 := cp.cpuPercentX1000Cur.Load()

	return fmt.Sprintf("%s RAM (RES):%4d MiB, CPU:%4.1f%%", cp.name,
		curRSSMib, float64(curCPUPercentageX1000)/1000.0)
}

// GetTotalConsumption returns total resource consumption since start of process
func (cp *childProcessCollector) GetTotalConsumption() *ResourceConsumption {
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

func containsConfig(s []string) bool {
	for _, a := range s {
		if a == "--config" {
			return true
		}
	}
	return false
}
