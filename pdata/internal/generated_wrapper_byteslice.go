// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package internal

type ByteSlice struct {
	orig  *[]byte
	state *State
}

func GetOrigByteSlice(ms ByteSlice) *[]byte {
	return ms.orig
}

func GetByteSliceState(ms ByteSlice) *State {
	return ms.state
}

func NewByteSlice(orig *[]byte, state *State) ByteSlice {
	return ByteSlice{orig: orig, state: state}
}

func FillTestByteSlice(tv ByteSlice) {
}

func GenerateTestByteSlice() ByteSlice {
	state := StateMutable
	return ByteSlice{&[]byte{}, &state}
}
