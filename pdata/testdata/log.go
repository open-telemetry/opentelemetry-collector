// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

var logTimestamp = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC))

func GenerateLogs(count int) plog.Logs {
	ld := plog.NewLogs()
	initResource(ld.ResourceLogs().AppendEmpty().Resource())
	logs := ld.ResourceLogs().At(0).ScopeLogs().AppendEmpty().LogRecords()
	logs.EnsureCapacity(count)
	for i := range count {
		switch i % 2 {
		case 0:
			fillLogOne(logs.AppendEmpty())
		case 1:
			fillLogTwo(logs.AppendEmpty())
		}
	}
	return ld
}

func fillLogOne(log plog.LogRecord) {
	log.SetTimestamp(logTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(plog.SeverityNumberInfo)
	log.SetSeverityText("Info")
	log.SetSpanID([8]byte{0x01, 0x02, 0x04, 0x08})
	log.SetTraceID([16]byte{0x08, 0x04, 0x02, 0x01})

	attrs := log.Attributes()
	attrs.PutStr("app", "server")
	attrs.PutInt("instance_num", 1)

	log.Body().SetStr("This is a log message")
}

func fillLogTwo(log plog.LogRecord) {
	log.SetTimestamp(logTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(plog.SeverityNumberInfo)
	log.SetSeverityText("Info")

	attrs := log.Attributes()
	attrs.PutStr("customer", "acme")
	attrs.PutStr("env", "dev")

	log.Body().SetStr("something happened")
}
