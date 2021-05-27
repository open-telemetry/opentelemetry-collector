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

// +build windows

package perfcounters

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/leoluk/perflib_exporter/perflib"

	"go.opentelemetry.io/collector/internal/processor/filterset"
)

const totalInstanceName = "_Total"

// PerfCounterScraper scrapes performance counter data.
type PerfCounterScraper interface {
	// start initializes the PerfCounterScraper so that subsequent calls
	// to scrape will return performance counter data for the specified set.
	// of objects
	Initialize(objects ...string) error
	// scrape returns performance data for the initialized objects.
	Scrape() (PerfDataCollection, error)
}

// PerfLibScraper is an implementation of PerfCounterScraper that uses
// perflib to scrape performance counter data.
type PerfLibScraper struct {
	objectIndices string
}

func (p *PerfLibScraper) Initialize(objects ...string) error {
	// "Counter 009" reads perf counter names in English.
	// This is always present regardless of the OS language.
	nameTable := perflib.QueryNameTable("Counter 009")

	// lookup object indices from name table
	objectIndicesMap := map[uint32]struct{}{}
	for _, name := range objects {
		index := nameTable.LookupIndex(name)
		if index == 0 {
			return fmt.Errorf("Failed to retrieve perf counter object %q", name)
		}

		objectIndicesMap[index] = struct{}{}
	}

	// convert to a space-separated string
	objectIndicesSlice := make([]string, 0, len(objectIndicesMap))
	for k := range objectIndicesMap {
		objectIndicesSlice = append(objectIndicesSlice, strconv.Itoa(int(k)))
	}
	p.objectIndices = strings.Join(objectIndicesSlice, " ")
	return nil
}

func (p *PerfLibScraper) Scrape() (PerfDataCollection, error) {
	objects, err := perflib.QueryPerformanceData(p.objectIndices)
	if err != nil {
		return nil, err
	}

	indexed := make(map[string]*perflib.PerfObject)
	for _, obj := range objects {
		indexed[obj.Name] = obj
	}

	return perfDataCollection{perfObject: indexed}, nil
}

// PerfDataCollection represents a collection of perf counter data.
type PerfDataCollection interface {
	// GetObject returns the perf counter data associated with the specified object,
	// or returns an error if no data exists for this object name.
	GetObject(objectName string) (PerfDataObject, error)
}

type perfDataCollection struct {
	perfObject map[string]*perflib.PerfObject
}

func (p perfDataCollection) GetObject(objectName string) (PerfDataObject, error) {
	obj, ok := p.perfObject[objectName]
	if !ok {
		return nil, fmt.Errorf("Unable to find object %q", objectName)
	}

	return perfDataObject{obj}, nil
}

// PerfDataCollection represents a collection of perf counter values
// and associated instances.
type PerfDataObject interface {
	// Filter filters the perf counter data to only retain data related to
	// relevant instances based on the supplied parameters.
	Filter(includeFS, excludeFS filterset.FilterSet, includeTotal bool)
	// GetValues returns the performance counter data associated with the specified
	// counters, or returns an error if any of the specified counter names do not
	// exist.
	GetValues(counterNames ...string) ([]*CounterValues, error)
}

type perfDataObject struct {
	*perflib.PerfObject
}

func (obj perfDataObject) Filter(includeFS, excludeFS filterset.FilterSet, includeTotal bool) {
	if includeFS == nil && excludeFS == nil && includeTotal {
		return
	}

	filteredDevices := make([]*perflib.PerfInstance, 0, len(obj.Instances))
	for _, device := range obj.Instances {
		if includeDevice(device.Name, includeFS, excludeFS, includeTotal) {
			filteredDevices = append(filteredDevices, device)
		}
	}
	obj.Instances = filteredDevices
}

func includeDevice(deviceName string, includeFS, excludeFS filterset.FilterSet, includeTotal bool) bool {
	if deviceName == totalInstanceName {
		return includeTotal
	}

	return (includeFS == nil || includeFS.Matches(deviceName)) &&
		(excludeFS == nil || !excludeFS.Matches(deviceName))
}

// CounterValues represents a set of perf counter values for a given instance.
type CounterValues struct {
	InstanceName string
	Values       map[string]int64
}

type counterIndex struct {
	index int
	name  string
}

func (obj perfDataObject) GetValues(counterNames ...string) ([]*CounterValues, error) {
	counterIndices := make([]counterIndex, 0, len(counterNames))
	for idx, counter := range obj.CounterDefs {
		// "Base" values give the value of a related counter that pdh.dll uses to compute the derived
		// value for this counter. We only care about raw values so ignore base values. See
		// https://docs.microsoft.com/en-us/windows/win32/perfctrs/retrieving-counter-data.
		if counter.IsBaseValue && !counter.IsNanosecondCounter {
			continue
		}

		for _, counterName := range counterNames {
			if counter.Name == counterName {
				counterIndices = append(counterIndices, counterIndex{index: idx, name: counter.Name})
				break
			}
		}
	}

	if len(counterIndices) < len(counterNames) {
		return nil, fmt.Errorf("Unable to find counters %q in object %q", missingCounterNames(counterNames, counterIndices), obj.Name)
	}

	values := make([]*CounterValues, len(obj.Instances))
	for i, instance := range obj.Instances {
		instanceValues := &CounterValues{InstanceName: instance.Name, Values: make(map[string]int64, len(counterIndices))}
		for _, counter := range counterIndices {
			instanceValues.Values[counter.name] = instance.Counters[counter.index].Value
		}
		values[i] = instanceValues
	}
	return values, nil
}

func missingCounterNames(counterNames []string, counterIndices []counterIndex) []string {
	matchedCounters := make(map[string]struct{}, len(counterIndices))
	for _, counter := range counterIndices {
		matchedCounters[counter.name] = struct{}{}
	}

	counters := make([]string, 0, len(counterNames)-len(matchedCounters))
	for _, counter := range counterNames {
		if _, ok := matchedCounters[counter]; !ok {
			counters = append(counters, counter)
		}
	}
	return counters
}
