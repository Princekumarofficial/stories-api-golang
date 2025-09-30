#!/bin/bash

# Generate Swagger documentation
echo "Generating Swagger documentation..."
swag init -g cmd/stories-service/main.go

# Fix the generated docs.go file by removing incompatible fields
echo "Fixing docs.go compatibility issues..."
sed -i '/LeftDelim:/d' docs/docs.go
sed -i '/RightDelim:/d' docs/docs.go

echo "Swagger docs generated and fixed successfully!"
echo "Access the docs at: http://localhost:8080/swagger/"
