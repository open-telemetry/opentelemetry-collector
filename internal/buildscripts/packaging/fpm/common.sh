#!/bin/bash

FPM_DIR="$( cd "$( dirname ${BASH_SOURCE[0]} )" && pwd )"

PKG_NAME="otel-collector"
PKG_VENDOR="OpenTelemetry Community"
PKG_MAINTAINER="OpenTelemetry Community <cncf-opentelemetry-community@lists.cncf.io>"
PKG_DESCRIPTION="OpenTelemetry Collector"
PKG_LICENSE="Apache 2.0"
PKG_URL="https://github.com/open-telemetry/opentelemetry-collector"
PKG_USER="otel"
PKG_GROUP="otel"

SERVICE_NAME="otel-collector"
PROCESS_NAME="otelcol"

SERVICE_PATH="$FPM_DIR/$SERVICE_NAME.service"
PREINSTALL_PATH="$FPM_DIR/preinstall.sh"
POSTINSTALL_PATH="$FPM_DIR/postinstall.sh"
PREUNINSTALL_PATH="$FPM_DIR/preuninstall.sh"

install_pkg() {
    local pkg_path="$1"
    local pkg_base=$( basename "$pkg_path" )

    echo "Installing $pkg_base ..."
    docker cp "$pkg_path" $image_name:/tmp/$pkg_base
    if [[ "${pkg_base##*.}" = "deb" ]]; then
        $docker_exec dpkg -i /tmp/$pkg_base
    else
        $docker_exec rpm -ivh /tmp/$pkg_base
    fi
}

uninstall_pkg() {
    local pkg_type="$1"
    local pkg_name="${2:-"$PKG_NAME"}"

    echo "Uninstalling $pkg_name ..."
    if [[ "$pkg_type" = "deb" ]]; then
        $docker_exec dpkg -r $pkg_name
    else
        $docker_exec rpm -e $pkg_name
    fi
}