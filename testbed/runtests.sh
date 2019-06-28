#!/bin/bash

set -e

cd tests

SED="sed --unbuffered"

PASS_COLOR=$(printf "\033[32mPASS\033[0m")
FAIL_COLOR=$(printf "\033[31mFAIL\033[0m")
TEST_COLORIZE="${SED} 's/PASS/${PASS_COLOR}/' | ${SED} 's/FAIL/${FAIL_COLOR}/'"

TESTBED_CONFIG=local.yaml go test -v 2>&1 | tee results/testoutput.log | bash -c "${TEST_COLORIZE}"

testStatus=${PIPESTATUS[0]}

mkdir -p results/junit
go-junit-report < results/testoutput.log > results/junit/results.xml

bash -c "cat results/TESTRESULTS.md | ${TEST_COLORIZE}"

exit ${testStatus}
