#!/bin/bash

# Run tests with coverage
echo "Running tests with coverage..."
go test -v -race -coverprofile=coverage.out ./...

# Display coverage summary
echo -e "\nCoverage summary:"
go tool cover -func=coverage.out

# Generate HTML report
echo -e "\nGenerating HTML coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo -e "\nCoverage report generated: coverage.html"
echo "Open coverage.html in your browser to view the detailed report" 
