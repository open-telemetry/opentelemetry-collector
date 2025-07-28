#!/bin/bash

# Script to capture deadlock stack trace from memory queue test

echo "Starting queue deadlock capture..."
cd /home/jmacd/src/otel/collector/exporter/exporterhelper/internal/queue

# Start the test in background and capture PID
echo "Building test binary..."
go test -c .

echo "Starting test in background..."
./queue.test -test.v -test.run TestMemoryQueueBlockOnOverflowDeadlock 2> queue_deadlock_err 1> queue_deadlock_out &
TEST_PID=$!

echo "Test started with PID: $TEST_PID"

# Wait for the test to reach deadlock state
echo "Waiting 8 seconds for deadlock to occur..."
sleep 8

# Send SIGABRT to get stack trace
echo "Sending SIGABRT to capture stack trace..."
kill -ABRT $TEST_PID

# Wait a moment for the signal to be processed
sleep 2

# Kill the process
kill -9 $TEST_PID 2>/dev/null

echo "=== STDERR Output (contains stack trace) ==="
cat queue_deadlock_err

echo "=== STDOUT Output ==="
cat queue_deadlock_out
