// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows && win32service

package otelcol

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	collectorServiceName = "otelcorecol"
)

// Test the collector as a Windows service.
// The test assumes that the service and respective event source are already created.
//
// To test locally:
// * Build the binary:
//   - make otelcorecol
//
// * Install the Windows service
//   - New-Service -Name "otelcorecol" -StartupType "Manual" -BinaryPathName "${PWD}\bin\otelcorecol_windows_$(go env GOARCH) --config ${PWD}\examples\local\otel-config.yaml"
//
// * Create event log source
//   - eventcreate.exe /t information /id 1 /l application /d "Creating event provider for 'otelcorecol'" /so otelcorecol
//
// The test also must be executed with administrative privileges.
func TestCollectorAsService(t *testing.T) {
	collector_executable, err := filepath.Abs(filepath.Join("..", "bin", fmt.Sprintf("otelcorecol_windows_%s", runtime.GOARCH)))
	require.NoError(t, err)
	_, err = os.Stat(collector_executable)
	require.NoError(t, err)

	scm, err := mgr.Connect()
	require.NoError(t, err)
	defer scm.Disconnect()

	service, err := scm.OpenService(collectorServiceName)
	require.NoError(t, err)
	defer service.Close()

	tests := []struct {
		name               string
		configFile         string
		expectStartFailure bool
		customSetup        func(*testing.T)
		customValidation   func(*testing.T)
	}{
		{
			name:       "Default",
			configFile: filepath.Join("..", "examples", "local", "otel-config.yaml"),
		},
		{
			name:               "ConfigFileNotFound",
			configFile:         filepath.Join(".", "non", "existent", "otel-config.yaml"),
			expectStartFailure: true,
		},
		{
			name:       "LogToFile",
			configFile: filepath.Join(".", "testdata", "otelcol-log-to-file.yaml"),
			customSetup: func(t *testing.T) {
				// Create the folder and clean the log file if it exists
				programDataPath := os.Getenv("ProgramData")
				logsPath := filepath.Join(programDataPath, "OpenTelemetry", "Collector", "Logs")
				err := os.MkdirAll(logsPath, os.ModePerm)
				require.NoError(t, err)

				logFilePath := filepath.Join(logsPath, "otelcol.log")
				err = os.Remove(logFilePath)
				if err != nil && !os.IsNotExist(err) {
					require.NoError(t, err)
				}
			},
			customValidation: func(t *testing.T) {
				// Check that the log file was created
				programDataPath := os.Getenv("ProgramData")
				logsPath := filepath.Join(programDataPath, "OpenTelemetry", "Collector", "Logs")
				logFilePath := filepath.Join(logsPath, "otelcol.log")
				fileinfo, err := os.Stat(logFilePath)
				require.NoError(t, err)
				require.NotEmpty(t, fileinfo.Size())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceConfig, err := service.Config()
			require.NoError(t, err)

			// Setup the command line to launch the collector as a service
			fullConfigPath, err := filepath.Abs(tt.configFile)
			require.NoError(t, err)

			serviceConfig.BinaryPathName = fmt.Sprintf("\"%s\" --config \"%s\"", collector_executable, fullConfigPath)
			err = service.UpdateConfig(serviceConfig)
			require.NoError(t, err)

			if tt.customSetup != nil {
				tt.customSetup(t)
			}

			startTime := time.Now()

			err = service.Start()
			require.NoError(t, err)

			expectedState := svc.Running
			if tt.expectStartFailure {
				expectedState = svc.Stopped
			} else {
				defer func() {
					_, err = service.Control(svc.Stop)
					require.NoError(t, err)

					require.Eventually(t, func() bool {
						status, _ := service.Query()
						return status.State == svc.Stopped
					}, 10*time.Second, 500*time.Millisecond)
				}()
			}

			// Wait for the service to reach the expected state
			require.Eventually(t, func() bool {
				status, _ := service.Query()
				return status.State == expectedState
			}, 10*time.Second, 500*time.Millisecond)

			if tt.customValidation != nil {
				tt.customValidation(t)
			} else {
				// Read the events from the otelcorecol source and check that they were emitted after the service
				// command started. This is a simple validation that the messages are being logged on the
				// Windows event log.
				cmd := exec.Command("wevtutil.exe", "qe", "Application", "/c:1", "/rd:true", "/f:RenderedXml", "/q:*[System[Provider[@Name='otelcorecol']]]")
				out, err := cmd.CombinedOutput()
				require.NoError(t, err)

				var e Event
				require.NoError(t, xml.Unmarshal([]byte(out), &e))

				eventTime, err := time.Parse("2006-01-02T15:04:05.9999999Z07:00", e.System.TimeCreated.SystemTime)
				require.NoError(t, err)

				require.True(t, eventTime.After(startTime.In(time.UTC)))
			}
		})
	}
}

// Helper types to read the XML events from the event log using wevtutil
type Event struct {
	XMLName xml.Name `xml:"Event"`
	System  System   `xml:"System"`
	Data    string   `xml:"EventData>Data"`
}

type System struct {
	Provider      Provider    `xml:"Provider"`
	EventID       int         `xml:"EventID"`
	Version       int         `xml:"Version"`
	Level         int         `xml:"Level"`
	Task          int         `xml:"Task"`
	Opcode        int         `xml:"Opcode"`
	Keywords      string      `xml:"Keywords"`
	TimeCreated   TimeCreated `xml:"TimeCreated"`
	EventRecordID int         `xml:"EventRecordID"`
	Execution     Execution   `xml:"Execution"`
	Channel       string      `xml:"Channel"`
	Computer      string      `xml:"Computer"`
}

type Provider struct {
	Name string `xml:"Name,attr"`
}

type TimeCreated struct {
	SystemTime string `xml:"SystemTime,attr"`
}

type Execution struct {
	ProcessID string `xml:"ProcessID,attr"`
	ThreadID  string `xml:"ThreadID,attr"`
}
