// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplepkg

import "errors"

const (
	ProtocolHTTP int = iota
	ProtocolTCP
	ProtocolSMTP
	ProtocolFTP
)

func (p *Protocol) UnmarshalText(text []byte) error {
	str := string(text)
	switch str {
	case "http":
		*p = Protocol(ProtocolHTTP)
	case "tcp":
		*p = Protocol(ProtocolTCP)
	case "smtp":
		*p = Protocol(ProtocolSMTP)
	case "ftp":
		*p = Protocol(ProtocolFTP)
	default:
		return errors.New("unknown protocol: " + str)
	}
	return nil
}
