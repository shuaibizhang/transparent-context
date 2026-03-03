#!/bin/bash
set -e

mkdir -p bin

# Build
echo "Building services..."
go build -o bin/service_c ./service_c
go build -o bin/service_b ./service_b
go build -o bin/service_a ./service_a

# Run Service C
echo "Starting Service C..."
./bin/service_c &
PID_C=$!

# Run Service B
echo "Starting Service B..."
./bin/service_b &
PID_B=$!

# Ensure cleanup on exit
trap "kill $PID_C $PID_B" EXIT

# Wait for services to start
sleep 5

# Run Service A
echo "Running Service A..."
./bin/service_a
