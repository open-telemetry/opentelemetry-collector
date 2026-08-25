// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

package cgroups

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCGroupParamPath(t *testing.T) {
	cgroup := NewCGroup("/sys/fs/cgroup/cpu")
	assert.Equal(t, "/sys/fs/cgroup/cpu", cgroup.Path())
	assert.Equal(t, "/sys/fs/cgroup/cpu/cpu.cfs_quota_us", cgroup.ParamPath("cpu.cfs_quota_us"))
}

func TestCGroupReadFirstLine(t *testing.T) {
	testTable := []struct {
		name            string
		paramName       string
		expectedContent string
		shouldHaveError bool
	}{
		{
			name:            "cpu",
			paramName:       "cpu.cfs_period_us",
			expectedContent: "100000",
			shouldHaveError: false,
		},
		{
			name:            "absent",
			paramName:       "cpu.stat",
			expectedContent: "",
			shouldHaveError: true,
		},
		{
			name:            "empty",
			paramName:       "cpu.cfs_quota_us",
			expectedContent: "",
			shouldHaveError: true,
		},
	}

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, tt.name)
		cgroup := NewCGroup(cgroupPath)

		content, err := cgroup.readFirstLine(tt.paramName)
		assert.Equal(t, tt.expectedContent, content, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

func TestCGroupReadInt(t *testing.T) {
	testTable := []struct {
		name            string
		paramName       string
		expectedValue   int64
		shouldHaveError bool
	}{
		{
			name:            "cpu",
			paramName:       "cpu.cfs_period_us",
			expectedValue:   100000,
			shouldHaveError: false,
		},
		{
			name:            "empty",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
		{
			name:            "invalid",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
		{
			name:            "absent",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
	}

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, tt.name)
		cgroup := NewCGroup(cgroupPath)

		value, err := cgroup.readInt(tt.paramName)
		assert.Equal(t, tt.expectedValue, value, "%s/%s", tt.name, tt.paramName)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
