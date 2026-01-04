set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# List available recipes. This is the default target.
default: help

# Display all documented targets.
help:
  @just --list

# Generate the SumUp client from the OpenAPI specification.
generate:
  go -C codegen run ./... --spec ../openapi.json --output ../src/SumUp --namespace SumUp

# Format the entire solution using dotnet-format.
fmt:
  dotnet format SumUp.sln

# Build the solution.
build:
  dotnet build SumUp.sln

# Execute the unit test suite.
test:
  DOTNET_ROLL_FORWARD=Major dotnet test SumUp.sln
