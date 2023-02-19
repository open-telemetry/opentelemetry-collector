// Copyright The OpenTelemetry Authors
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

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package internal

type UInt64Slice struct {
	orig *[]uint64
}

func GetOrigUInt64Slice(ms UInt64Slice) *[]uint64 {
	return ms.orig
}

func NewUInt64Slice(orig *[]uint64) UInt64Slice {
	return UInt64Slice{orig: orig}
}

func CopyOrigUInt64Slice(dst, src []uint64) []uint64 {
	dst = dst[:0]
	return append(dst, src...)
}
