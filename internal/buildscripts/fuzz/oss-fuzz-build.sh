#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# This script is used by OSS-Fuzz to build and run the Collectors fuzz tests continuously.
# Limas OSS-Fuzz integration can be found here: https://github.com/google/oss-fuzz/tree/master/projects/opentelemetry
# Modify https://github.com/google/oss-fuzz/blob/master/projects/opentelemetry/project.yaml for access management to the Collectors OSS-Fuzz dashboard.

# shellcheck disable=SC2164
cd "${SRC}"/go-118-fuzz-build
go build .
mv go-118-fuzz-build /root/go/bin/

# shellcheck disable=SC2164
cd "${SRC}"/opentelemetry-collector
# shellcheck disable=SC2164
cd receiver/otlpreceiver
printf "package otlpreceiver \nimport _ \"github.com/AdamKorcz/go-118-fuzz-build/testing\"\n" > ./fuzz-register.go
go mod tidy
go mod edit -replace github.com/AdamKorcz/go-118-fuzz-build="${SRC}"/go-118-fuzz-build
go mod tidy
compile_native_go_fuzzer go.opentelemetry.io/collector/receiver/otlpreceiver FuzzReceiverHandlers FuzzReceiverHandlers
# shellcheck disable=SC2164
cd "${SRC}"/opentelemetry-collector

# shellcheck disable=SC2164
cd pdata
printf "package ptrace\nimport _ \"github.com/AdamKorcz/go-118-fuzz-build/testing\"\n" > "${SRC}"/opentelemetry-collector/pdata/ptrace/register.go
go mod tidy
go mod edit -replace github.com/AdamKorcz/go-118-fuzz-build="${SRC}"/go-118-fuzz-build
go mod tidy
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/plog FuzzUnmarshalJsonLogs FuzzUnmarshalJsonLogs_plogs
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/plog FuzzUnmarshalPBLogs FuzzUnmarshalPBLogs_plogs
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/plog/plogotlp FuzzRequestUnmarshalJSON FuzzRequestUnmarshalJSON_plogotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/plog/plogotlp FuzzResponseUnmarshalJSON FuzzResponseUnmarshalJSON_plogotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/plog/plogotlp FuzzRequestUnmarshalProto FuzzRequestUnmarshalProto_plogotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/plog/plogotlp FuzzResponseUnmarshalProto FuzzResponseUnmarshalProto_plogotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/pmetric FuzzUnmarshalMetrics FuzzUnmarshalMetrics_pmetric
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp FuzzRequestUnmarshalJSON FuzzRequestUnmarshalJSON_pmetricotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp FuzzResponseUnmarshalJSON FuzzResponseUnmarshalJSON_pmetricotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp FuzzRequestUnmarshalProto FuzzRequestUnmarshalProto_pmetricotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp FuzzResponseUnmarshalProto FuzzResponseUnmarshalProto_pmetricotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/ptrace FuzzUnmarshalPBTraces FuzzUnmarshalPBTraces_ptrace
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/ptrace FuzzUnmarshalJSONTraces FuzzUnmarshalJSONTraces_ptrace
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp FuzzRequestUnmarshalJSON FuzzRequestUnmarshalJSON_ptraceotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp FuzzResponseUnmarshalJSON FuzzResponseUnmarshalJSON_ptraceotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp FuzzRequestUnmarshalProto FuzzRequestUnmarshalProto_ptraceotlp
compile_native_go_fuzzer go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp FuzzResponseUnmarshalProto FuzzResponseUnmarshalProto_ptraceotlp

cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzUnmarshalJsonLogs_plogs.dict
cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzRequestUnmarshalJSON_plogotlp.dict
cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzResponseUnmarshalJSON_plogotlp.dict
cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzUnmarshalMetrics_pmetric.dict
cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzRequestUnmarshalJSON_pmetricotlp.dict
cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzResponseUnmarshalJSON_pmetricotlp.dict
cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzUnmarshalJSONTraces_ptrace.dict
cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzRequestUnmarshalJSON_ptraceotlp.dict
cp "${SRC}"/opentelemetry-collector/internal/buildscripts/fuzz/json_dict "${OUT}"/FuzzResponseUnmarshalJSON_ptraceotlp.dict

zip "${OUT}"/FuzzUnmarshalJsonLogs_plogs_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
zip "${OUT}"/FuzzRequestUnmarshalJSON_plogotlp_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
zip "${OUT}"/FuzzResponseUnmarshalJSON_plogotlp_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
zip "${OUT}"/FuzzUnmarshalMetrics_pmetric_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
zip "${OUT}"/FuzzRequestUnmarshalJSON_pmetricotlp_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
zip "${OUT}"/FuzzResponseUnmarshalJSON_pmetricotlp_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
zip "${OUT}"/FuzzUnmarshalJSONTraces_ptrace_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
zip "${OUT}"/FuzzRequestUnmarshalJSON_ptraceotlp_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
zip "${OUT}"/FuzzResponseUnmarshalJSON_ptraceotlp_seed_corpus.zip "${SRC}"/go-fuzz-corpus/json/corpus/*
