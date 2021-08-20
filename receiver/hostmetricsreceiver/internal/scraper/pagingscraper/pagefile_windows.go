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

package pagingscraper

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modPsapi    = windows.NewLazySystemDLL("psapi.dll")

	procGetNativeSystemInfo = modKernel32.NewProc("GetNativeSystemInfo")
	procEnumPageFilesW      = modPsapi.NewProc("EnumPageFilesW")
)

type systemInfo struct {
	wProcessorArchitecture      uint16
	wReserved                   uint16
	dwPageSize                  uint32
	lpMinimumApplicationAddress uintptr
	lpMaximumApplicationAddress uintptr
	dwActiveProcessorMask       uintptr
	dwNumberOfProcessors        uint32
	dwProcessorType             uint32
	dwAllocationGranularity     uint32
	wProcessorLevel             uint16
	wProcessorRevision          uint16
}

func getPageSize() uint64 {
	var sysInfo systemInfo
	procGetNativeSystemInfo.Call(uintptr(unsafe.Pointer(&sysInfo)))
	return uint64(sysInfo.dwPageSize)
}

type pageFileData struct {
	name       string
	usedPages  uint64
	totalPages uint64
}

// system type as defined in https://docs.microsoft.com/en-us/windows/win32/api/psapi/ns-psapi-enum_page_file_information
type enumPageFileInformation struct {
	cb         uint32
	reserved   uint32
	totalSize  uint64
	totalInUse uint64
	peakUsage  uint64
}

func getPageFileStats() ([]*pageFileData, error) {
	// the following system call invokes the supplied callback function once for each page file before returning
	// see https://docs.microsoft.com/en-us/windows/win32/api/psapi/nf-psapi-enumpagefilesw
	var pageFiles []*pageFileData
	result, _, _ := procEnumPageFilesW.Call(windows.NewCallback(pEnumPageFileCallbackW), uintptr(unsafe.Pointer(&pageFiles)))
	if result == 0 {
		return nil, windows.GetLastError()
	}

	return pageFiles, nil
}

// system callback as defined in https://docs.microsoft.com/en-us/windows/win32/api/psapi/nc-psapi-penum_page_file_callbackw
func pEnumPageFileCallbackW(pageFiles *[]*pageFileData, enumPageFileInfo *enumPageFileInformation, lpFilenamePtr *[syscall.MAX_LONG_PATH]uint16) *bool {
	pageFileName := syscall.UTF16ToString((*lpFilenamePtr)[:])

	pfData := &pageFileData{
		name:       pageFileName,
		usedPages:  enumPageFileInfo.totalInUse,
		totalPages: enumPageFileInfo.totalSize,
	}

	*pageFiles = append(*pageFiles, pfData)

	// return true to continue enumerating page files
	ret := true
	return &ret
}
