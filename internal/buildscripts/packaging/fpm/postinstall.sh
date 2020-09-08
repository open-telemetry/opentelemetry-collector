#!/bin/sh

if command -v systemctl >/dev/null 2>&1; then
    systemctl enable otel-collector.service
    if [ -f /etc/otel-collector/config.yaml ]; then
        systemctl start otel-collector.service
    fi
fi