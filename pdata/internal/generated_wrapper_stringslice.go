// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package internal

type StringSlice struct {
	orig  *[]string
	state *State
}

func GetOrigStringSlice(ms StringSlice) *[]string {
	return ms.orig
}

func GetStringSliceState(ms StringSlice) *State {
	return ms.state
}

func NewStringSlice(orig *[]string, state *State) StringSlice {
	return StringSlice{orig: orig, state: state}
}

func CopyOrigStringSlice(dst, src []string) []string {
	dst = dst[:0]
	return append(dst, src...)
}

func FillTestStringSlice(tv StringSlice) {
}

func GenerateTestStringSlice() StringSlice {
	state := StateMutable
	var orig []string = nil

	return StringSlice{&orig, &state}
}
