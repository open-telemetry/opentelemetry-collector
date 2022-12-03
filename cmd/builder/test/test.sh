#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
export WORKSPACE_DIR=$( cd -- "$( dirname $(dirname $(dirname -- "${SCRIPT_DIR}")) )" &> /dev/null && pwd )

GOBIN=$(go env GOBIN)
if [[ "$GO" == "" ]]; then
    GOBIN=$(which go)
fi

if [[ "$GOBIN" == "" ]]; then
    echo "Could not determine which Go binary to use."
    exit 1
fi

echo "Using ${GOBIN} to compile the distributions."

test_build_config() {
    local test="$1"
    local build_config="$2"

    out="${base}/${test}"
    mkdir -p "${out}"
    if [ $? != 0 ]; then
        echo "❌ FAIL ${test}. Failed to create test directory for the test. Aborting tests."
        exit 2
    fi

    echo "Starting test '${test}' at `date`" >> "${out}/test.log"

    final_build_config=$(basename ${build_config})
    envsubst < "$build_config" > "${out}/${final_build_config}"
    go run . --go "${GOBIN}" --config "${out}/${final_build_config}" --output-path "${out}" --name otelcol-built-test > "${out}/builder.log" 2>&1

    if [ $? != 0 ]; then
        echo "❌ FAIL ${test}. Failed to compile the test ${test}. Build logs:"
        cat "${out}/builder.log"
        failed=true
        return
    fi

    if [ ! -f "${out}/otelcol-built-test" ]; then
        echo "❌ FAIL ${test}. Binary not found for ${test} at '${out}/otelcol-built-test'. Build logs:"
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

    "${out}/otelcol-built-test" --config "./test/${test}.otel.yaml" > "${out}/otelcol.log" 2>&1 &
    pid=$!

    retries=0
    while true
    do
        kill -0 "${pid}" >/dev/null 2>&1
        if [ $? != 0 ]; then
            echo "❌ FAIL ${test}. The OpenTelemetry Collector isn't running. Startup log:"
            cat "${out}/otelcol.log"
            failed=true
            break
        fi

        # Since the content of the servicez page depend on which extensions are
        # built into the collector, we depend only on the zpages extension
        # being present and serving something.
        curl --fail --silent --output /dev/null http://localhost:55679/debug/servicez
        if [ $? == 0 ]; then
            echo "✅ PASS ${test}"

            kill "${pid}"
            if [ $? != 0 ]; then
                echo "Failed to stop the running instance for test ${test}. Return code: $? . Skipping tests."
                exit 4
            fi
            break
        fi

        echo "Server still unavailable for test '${test}'" >> "${out}/test.log"

        let "retries++"
        if [ "$retries" -gt "$max_retries" ]; then
            echo "❌ FAIL ${test}. Server wasn't up after about 5s."
            failed=true

            kill "${pid}"
            if [ $? != 0 ]; then
                echo "Failed to stop the running instance for test ${test}. Return code: $? . Skipping tests."
                exit 8
            fi
            break
        fi
        sleep 0.1s
    done

    echo "Stopping server for '${test}' (pid: ${pid})" >> "${out}/test.log"

}

# each attempt pauses for 100ms before retrying
max_retries=50

tests="core"

base=`mktemp -d`
echo "Running the tests in ${base}"

failed=false

for test in $tests
do
    test_build_config "$test" "./test/${test}.builder.yaml"
done

if [[ "$failed" == "true" ]]; then
    exit 1
fi
