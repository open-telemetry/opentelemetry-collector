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

package filesystemscraper

import (
	"fmt"

	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

// Config relating to FileSystem Metric Scraper.
type Config struct {
	internal.ConfigSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// IncludeDevices specifies a filter on the devices, filesystem types or mount points that should be included from the generated metrics.
	IncludeDevices DeviceMatchConfig `mapstructure:"include_devices"`
	// ExcludeDevices specifies a filter on the devices, filesystem types or mount points that should be excluded from the generated metrics.
	ExcludeDevices DeviceMatchConfig `mapstructure:"exclude_devices"`

	// IncludeFSTypes specifies a filter on the filesystem types that should be included from the generated metrics.
	IncludeFSTypes FSTypeMatchConfig `mapstructure:"include_fs_types"`
	// ExcludeFSTypes specifies a filter on the filesystem types points that should be excluded from the generated metrics.
	ExcludeFSTypes FSTypeMatchConfig `mapstructure:"exclude_fs_types"`

	// IncludeMountPoints specifies a filter on the mount points that should be included from the generated metrics.
	IncludeMountPoints MountPointMatchConfig `mapstructure:"include_mount_points"`
	// ExcludeMountPoints specifies a filter on the mount points that should be excluded from the generated metrics.
	ExcludeMountPoints MountPointMatchConfig `mapstructure:"exclude_mount_points"`
}

type DeviceMatchConfig struct {
	filterset.Config `mapstructure:",squash"`

	Devices []string `mapstructure:"devices"`
}

type FSTypeMatchConfig struct {
	filterset.Config `mapstructure:",squash"`

	FSTypes []string `mapstructure:"fs_types"`
}

type MountPointMatchConfig struct {
	filterset.Config `mapstructure:",squash"`

	MountPoints []string `mapstructure:"mount_points"`
}

func (cfg *Config) newFilter() (*fsFilter, error) {
	var err error
	filter := fsFilter{}

	filter.includeDevice, err = newIncludeFilterHelper(cfg.IncludeDevices.Devices, &cfg.IncludeDevices.Config, "device")
	if err != nil {
		return nil, err
	}

	filter.excludeDevice, err = newExcludeFilterHelper(cfg.ExcludeDevices.Devices, &cfg.ExcludeDevices.Config, "device")
	if err != nil {
		return nil, err
	}

	filter.includeFSType, err = newIncludeFilterHelper(cfg.IncludeFSTypes.FSTypes, &cfg.IncludeFSTypes.Config, "fs_type")
	if err != nil {
		return nil, err
	}

	filter.excludeFSType, err = newExcludeFilterHelper(cfg.ExcludeFSTypes.FSTypes, &cfg.ExcludeFSTypes.Config, "fs_type")
	if err != nil {
		return nil, err
	}

	filter.includeMountPoint, err = newIncludeFilterHelper(cfg.IncludeMountPoints.MountPoints, &cfg.IncludeMountPoints.Config, "mount_point")
	if err != nil {
		return nil, err
	}

	filter.excludeMountPoint, err = newExcludeFilterHelper(cfg.ExcludeMountPoints.MountPoints, &cfg.ExcludeMountPoints.Config, "mount_point")
	if err != nil {
		return nil, err
	}

	return &filter, nil
}

func newIncludeFilterHelper(items []string, filterSet *filterset.Config, typ string) (filterset.FilterSet, error) {
	return newFilterHelper(items, filterSet, "include", typ)
}

func newExcludeFilterHelper(items []string, filterSet *filterset.Config, typ string) (filterset.FilterSet, error) {
	return newFilterHelper(items, filterSet, "exclude", typ)
}

func newFilterHelper(items []string, filterSet *filterset.Config, typ string, filterType string) (filterset.FilterSet, error) {
	var err error
	var filter filterset.FilterSet
	if len(items) > 0 {
		filter, err = filterset.CreateFilterSet(items, filterSet)
		if err != nil {
			return nil, fmt.Errorf("error creating %s %s filters: %w", filterType, typ, err)
		}
	}
	return filter, nil
}
