#!/bin/bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
WORKSPACE_DIR=$( cd -- "$( dirname "$(dirname "$(dirname -- "${SCRIPT_DIR}")")" )" &> /dev/null && pwd )
export WORKSPACE_DIR

GOBIN=$(go env GOBIN)
if [[ "$GO" == "" ]]; then
    GOBIN=$(which go)
fi
export GOBIN

if [[ "$GOBIN" == "" ]]; then
    echo "Could not determine which Go binary to use."
    exit 1
fi

# Match the CGO_ENABLED=0 setting used when building distributions elsewhere
# (see cmd/builder/Makefile and the top-level Makefile).
export CGO_ENABLED=0

echo "Using ${GOBIN} to compile the distributions."

test_build_config() {
    local test="$1"
    local build_config="$2"

    out="${base}/${test}"
    if ! mkdir -p "${out}"; then
        echo "❌ FAIL ${test}. Failed to create test directory for the test. Aborting tests."
        exit 2
    fi

    echo "Starting test '${test}' at $(date)" >> "${out}/test.log"

    final_build_config=$(basename "${build_config}")
    go tool -modfile "${WORKSPACE_DIR}/internal/tools/go.mod" envsubst \
       -o "${out}/${final_build_config}" -i <(cat "$build_config" "$replaces")
    if ! go run . --config "${out}/${final_build_config}" --output-path "${out}" > "${out}/builder.log" 2>&1; then
        echo "❌ FAIL ${test}. Failed to compile the test ${test}. Build logs:"
        cat "${out}/builder.log"
        failed=true
        return
    fi

    if [ ! -f "${out}/${test}" ]; then
        echo "❌ FAIL ${test}. Binary not found for ${test} at '${out}/${test}'. Build logs:"
        cat "${out}/builder.log"
        failed=true
        return
    fi

    # start the distribution
    if [ ! -f "./test/${test}.otel.yaml" ]; then
        echo "❌ FAIL ${test}. Config file for ${test} not found at './test/${test}.otel.yaml'"
        failed=true
        return
    fi

    "${out}/${test}" --config "./test/${test}.otel.yaml" > "${out}/otelcol.log" 2>&1 &
    pid=$!

    # each attempt pauses for 100ms before retrying
    max_retries=50
    retries=0
    while true
    do
        if ! kill -0 "${pid}" >/dev/null 2>&1; then
            echo "❌ FAIL ${test}. The OpenTelemetry Collector isn't running. Startup log:"
            cat "${out}/otelcol.log"
            failed=true
            break
        fi

        # Since the content of the servicez page depend on which extensions are
        # built into the collector, we depend only on the zpages extension
        # being present and serving something.
        if curl --fail --silent --output /dev/null http://localhost:55679/debug/servicez; then
            echo "✅ PASS ${test}"

            kill "${pid}"
            ret=$?
            if [ $ret -ne 0 ]; then
                echo "Failed to stop the running instance for test ${test}. Return code: ${ret} . Skipping tests."
                exit 4
            fi
            break
        fi

        echo "Server still unavailable for test '${test}'" >> "${out}/test.log"

        ((retries++))
        if [ "$retries" -gt "$max_retries" ]; then
            echo "❌ FAIL ${test}. Server wasn't up after about 5s."
            failed=true

            kill "${pid}"
            ret=$?
            if [ $ret -ne 0 ]; then
                echo "Failed to stop the running instance for test ${test}. Return code: ${ret} . Skipping tests."
                exit 8
            fi
            break
        fi
        sleep 0.1s
    done

    echo "Stopping server for '${test}' (pid: ${pid})" >> "${out}/test.log"

}

# kill_tree PID stops a process and its descendants, so that the collector
# started by `make run` does not outlive make and keep its ports bound.
kill_tree() {
  local child
  for child in $(pgrep -P "$1" 2>/dev/null); do
    kill_tree "${child}"
  done
  kill "$1"
}

# test_init NAME [INIT_ARGS...] runs `ocb init` with the extra arguments in a
# folder named NAME, and checks that the generated collector starts.
test_init() {
  local name="$1"
  shift
  local start="$PWD"
  out="${base}/${name}"
  if ! mkdir -p "${out}"; then
      echo "❌ FAIL ${name}. Failed to create test directory for the test. Aborting tests."
      exit 2
  fi

  echo "Starting ${name} test at $(date)" >> "${out}/test.log"

  if ! go run . init --path "${out}" "$@" > "${out}/builder.log" 2>&1; then
      echo "❌ FAIL ${name}. Failed to compile the test. Build logs:"
      cat "${out}/builder.log"
      failed=true
      return
  fi

  go tool -modfile "${WORKSPACE_DIR}/internal/tools/go.mod" envsubst \
     -o "${out}/manifest.yaml" -i <(cat "${out}/manifest.yaml" "$replaces")

  cd "${out}" || exit 1
  make run > "${out}/otelcol.log" 2>&1 &
  pid=$!

  # each attempt pauses for 100ms before retrying
  max_retries=15000
  retries=0
  while true
  do
      if ! kill -0 "${pid}" >/dev/null 2>&1; then
          echo "❌ FAIL ${name}. The OpenTelemetry Collector isn't running. Startup log:"
          cat "${out}/otelcol.log"
          failed=true
          break
      fi

      if cat "${out}/otelcol.log" | grep "Everything is ready."; then
        echo "✅ PASS ${name}"

        kill_tree "${pid}"
        ret=$?
        if [ $ret -ne 0 ]; then
            echo "Failed to stop the running instance for test ${name}. Return code: ${ret} . Skipping tests."
            exit 4
        fi
        break
      fi

      echo "Server still unavailable for test '${name}'" >> "${out}/test.log"

      ((retries++))
      if [ "$retries" -gt "$max_retries" ]; then
          echo "❌ FAIL ${name}. Server wasn't up after about 5m."
          failed=true

          kill_tree "${pid}"
          ret=$?
          if [ $ret -ne 0 ]; then
              echo "Failed to stop the running instance for test ${name}. Return code: ${ret} . Skipping tests."
              exit 8
          fi
          break
      fi
      sleep 0.1s
  done

  echo "Stopping server for '${name}' (pid: ${pid})" >> "${out}/test.log"
  cd "${start}" || exit 1
}

tests="core"

base=$(mktemp -d)
echo "Running the tests in ${base}"

replaces="$base/replaces"
# Get path of all core modules, in sorted order
core_mods=$(cd ../.. && find . -type f -name "go.mod" -exec dirname {} \; | sort)
echo "replaces:" >> "$replaces"
for mod_path in $core_mods; do
    mod=${mod_path#"."} # remove initial dot
    echo "  - go.opentelemetry.io/collector$mod => \${WORKSPACE_DIR}$mod" >> "$replaces"
done
echo "Wrote replace statements to $replaces"

failed=false

for test in $tests
do
    test_build_config "$test" "./test/${test}.builder.yaml"
done

test_init init
test_init init-from-config --from-config "${PWD}/test/init-from-config.yaml"

if [[ "$failed" == "true" ]]; then
    exit 1
fi
