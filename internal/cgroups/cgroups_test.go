// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Keep the original Uber license.

// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

//go:build linux
// +build linux

package cgroups

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCGroups(t *testing.T) {
	cgroupsProcCGroupPath := filepath.Join(testDataProcPath, "cgroups", "cgroup")
	cgroupsProcMountInfoPath := filepath.Join(testDataProcPath, "cgroups", "mountinfo")

	testTable := []struct {
		subsys string
		path   string
	}{
		{_cgroupSubsysCPU, "/sys/fs/cgroup/cpu,cpuacct"},
		{_cgroupSubsysCPUAcct, "/sys/fs/cgroup/cpu,cpuacct"},
		{_cgroupSubsysCPUSet, "/sys/fs/cgroup/cpuset"},
		{_cgroupSubsysMemory, "/sys/fs/cgroup/memory/large"},
	}

	cgroups, err := NewCGroups(cgroupsProcMountInfoPath, cgroupsProcCGroupPath)
	assert.Equal(t, len(testTable), len(cgroups))
	assert.NoError(t, err)

	for _, tt := range testTable {
		cgroup, exists := cgroups[tt.subsys]
		assert.True(t, exists, "%q expected to present in `cgroups`", tt.subsys)
		assert.Equal(t, tt.path, cgroup.path, "%q expected for `cgroups[%q].path`, got %q", tt.path, tt.subsys, cgroup.path)
	}
}

func TestNewCGroupsWithErrors(t *testing.T) {
	testTable := []struct {
		mountInfoPath string
		cgroupPath    string
	}{
		{"non-existing-file", "/dev/null"},
		{"/dev/null", "non-existing-file"},
		{
			"/dev/null",
			filepath.Join(testDataProcPath, "invalid-cgroup", "cgroup"),
		},
		{
			filepath.Join(testDataProcPath, "invalid-mountinfo", "mountinfo"),
			"/dev/null",
		},
		{
			filepath.Join(testDataProcPath, "untranslatable", "mountinfo"),
			filepath.Join(testDataProcPath, "untranslatable", "cgroup"),
		},
	}

	for _, tt := range testTable {
		cgroups, err := NewCGroups(tt.mountInfoPath, tt.cgroupPath)
		assert.Nil(t, cgroups)
		assert.Error(t, err)
	}
}

func TestCGroupsCPUQuota(t *testing.T) {
	testTable := []struct {
		name            string
		expectedQuota   int64
		expectedDefined bool
		shouldHaveError bool
	}{
		{
			name:            "undefined",
			expectedQuota:   int64(-1.0),
			expectedDefined: false,
			shouldHaveError: false,
		},
	}

	cgroups := make(CGroups)

	quota, defined, err := cgroups.MemoryQuota()
	assert.Equal(t, int64(-1), quota, "nonexistent")
	assert.False(t, defined, "nonexistent")
	assert.NoError(t, err, "nonexistent")

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, tt.name)
		cgroups[_cgroupSubsysCPU] = NewCGroup(cgroupPath)

		quota, defined, err := cgroups.MemoryQuota()
		assert.Equal(t, tt.expectedQuota, quota, tt.name)
		assert.Equal(t, tt.expectedDefined, defined, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
