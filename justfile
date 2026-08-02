set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# List available recipes. This is the default target.
default: help

# Display all documented targets.
help:
  @just --list

# Generate the SumUp client from the OpenAPI specification.
generate:
  go -C codegen run ./... --spec ../openapi.json --output ../src/SumUp --namespace SumUp

# Generate the versioned C# code-sample catalog.
generate-codesamples output="code-samples.json":
  go -C codegen run . samples \
    --spec ../openapi.json \
    --sdk-version-file ../src/SumUp/SumUp.csproj \
    --output "{{ absolute_path(output) }}"

# Format the entire solution using dotnet-format.
fmt:
  dotnet format SumUp.sln

# Build the solution.
build:
  dotnet build SumUp.sln

# Execute the unit test suite.
test:
  DOTNET_ROLL_FORWARD=Major MSBUILDDISABLENODEREUSE=1 MSBUILDNOINPROCNODE=1 dotnet test SumUp.sln --disable-build-servers
