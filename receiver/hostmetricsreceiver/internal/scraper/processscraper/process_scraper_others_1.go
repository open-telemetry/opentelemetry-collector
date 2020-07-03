// Copyright 2020, OpenTelemetry Authors
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

// +build !windows

package processscraper

func getProcessExecutable(proc processHandle) (*executableMetadata, error) {
	name, err := proc.Name()
	if err != nil {
		return nil, err
	}

	exe, err := proc.Exe()
	if err != nil {
		return nil, err
	}

	executable := &executableMetadata{name: name, path: exe}
	return executable, nil
}

func getProcessCommand(proc processHandle) (*commandMetadata, error) {
	cmdline, err := proc.CmdlineSlice()
	if err != nil {
		return nil, err
	}

	var cmd string
	if len(cmdline) > 0 {
		cmd = cmdline[0]
	}

	command := &commandMetadata{command: cmd, commandLineSlice: cmdline}
	return command, nil
}
