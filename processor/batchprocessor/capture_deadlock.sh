#!/bin/bash

# Script to capture deadlock stack trace from batch processor test

echo "Starting deadlock capture..."
cd /home/jmacd/src/otel/collector/processor/batchprocessor

# Start the test in background and capture PID
echo "Starting test in background..."
go test -race -c .

./batchprocessor.test -test.v -test.run TestBatchProcessorSpansDeliveredEnforceBatchSize/TestBatchProcessorSpansDeliveredEnforceBatchSizeHelperWithPropagateErrors 2> deadlock_err 1> deadlock_out &
TEST_PID=$!

echo "Test started with PID: $TEST_PID"

# Wait for the test to reach deadlock state
echo "Waiting 5 seconds for deadlock to occur..."
sleep 5

# Send SIGABRT to get stack trace
echo "Sending SIGABRT to capture stack trace..."
kill -ABRT $TEST_PID

# Wait a moment for the signal to be processed
sleep 2

kill -9 $TEST_PID 2>/dev/null

cat deadlock_err
