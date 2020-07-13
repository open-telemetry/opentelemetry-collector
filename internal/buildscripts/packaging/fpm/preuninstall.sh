#!/bin/sh

if command -v systemctl >/dev/null 2>&1; then
    systemctl stop otel-collector.service
    systemctl disable otel-collector.service
fi