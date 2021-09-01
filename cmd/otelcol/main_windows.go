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

//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/svc"

	"go.opentelemetry.io/collector/service"
)

func run(set service.CollectorSettings) error {
	if useInteractiveMode, err := checkUseInteractiveMode(); err != nil {
		return err
	} else if useInteractiveMode {
		return runInteractive(set)
	} else {
		return runService(set)
	}
}

func checkUseInteractiveMode() (bool, error) {
	// If environment variable NO_WINDOWS_SERVICE is set with any value other
	// than 0, use interactive mode instead of running as a service. This should
	// be set in case running as a service is not possible or desired even
	// though the current session is not detected to be interactive
	if value, present := os.LookupEnv("NO_WINDOWS_SERVICE"); present && value != "0" {
		return true, nil
	}

	if isInteractiveSession, err := svc.IsAnInteractiveSession(); err != nil {
		return false, fmt.Errorf("failed to determine if we are running in an interactive session %w", err)
	} else {
		return isInteractiveSession, nil
	}
}

func runService(set service.CollectorSettings) error {
	// do not need to supply service name when startup is invoked through Service Control Manager directly
	if err := svc.Run("", service.NewWindowsService(set)); err != nil {
		return fmt.Errorf("failed to start service %w", err)
	}

	return nil
}
