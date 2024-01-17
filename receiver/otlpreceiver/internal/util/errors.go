// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package util // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/util"

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

func GetStatusFromError(err error) *status.Status {
	s, ok := status.FromError(err)
	if !ok {
		code := codes.Unavailable
		if consumererror.IsPermanent(err) {
			code = codes.InvalidArgument
		}
		s = status.New(code, err.Error())
	}
	return s
}
