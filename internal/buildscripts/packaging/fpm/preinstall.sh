#!/bin/sh

getent passwd otel >/dev/null || useradd --system --user-group --no-create-home --shell /sbin/nologin otel