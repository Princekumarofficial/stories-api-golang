#!/bin/bash

echo "Building Stories Service..."
go build -o bin/stories-service ./cmd/stories-service

echo "Building Ephemeral Worker..."
go build -o bin/ephemeral-worker ./cmd/ephemeral-worker

echo "Build completed successfully!"
echo "Run the services with:"
echo "  ./bin/stories-service (for the main API)"
echo "  ./bin/ephemeral-worker (for the worker)"
