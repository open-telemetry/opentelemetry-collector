// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package iruntime // import "go.opentelemetry.io/collector/internal/memorylimiter/iruntime"

import "go.opentelemetry.io/collector/internal/memorylimiter/cgroups"

// unlimitedMemorySize defines the bytes size when memory limit is not set
// for the container and process with cgroups
const unlimitedMemorySize = 9223372036854771712

// TotalMemory returns total available memory.
// This implementation is meant for linux and uses cgroups to determine available memory.
func TotalMemory() (uint64, error) {
	var memoryQuota int64
	var defined bool
	var err error

	isV2, err := cgroups.IsCGroupV2()
	if err != nil {
		return 0, err
	}

	if isV2 {
		memoryQuota, defined, err = cgroups.MemoryQuotaV2()
		if err != nil {
			return 0, err
		}
	} else {
		cgv1, err := cgroups.NewCGroupsForCurrentProcess()
		if err != nil {
			return 0, err
		}
		memoryQuota, defined, err = cgv1.MemoryQuota()
		if err != nil {
			return 0, err
		}
	}

	// If memory is not defined or is set to unlimitedMemorySize (v1 unset),
	// we fallback to /proc/meminfo.
	if memoryQuota == unlimitedMemorySize || !defined {
		totalMem, err := readMemInfo()
		if err != nil {
			return 0, err
		}
		return totalMem, nil
	}

	return uint64(memoryQuota), nil
}
