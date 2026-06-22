$ErrorActionPreference = "Stop"

# Navigate to the root directory of the microservice
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location "$ScriptDir\.."

Write-Host "Generating OpenAPI documentation using Swaggo..." -ForegroundColor Cyan

# Run swag init. It parses the annotations in src/main.go and controllers.
# It outputs swagger.json and swagger.yaml to the docs/ directory.
go run github.com/swaggo/swag/cmd/swag@latest init -g src/main.go --output docs --outputTypes json,yaml

# Create the .agents directory if it doesn't exist
if (-not (Test-Path ".agents")) {
    New-Item -ItemType Directory -Path ".agents" | Out-Null
}

# Copy/rename the generated JSON spec to the requested location in .agents
Copy-Item -Path "docs/swagger.json" -Destination "openapi.json" -Force

Write-Host "OpenAPI contract successfully generated:" -ForegroundColor Green
Write-Host "- docs/swagger.json"
Write-Host "- docs/swagger.yaml"
Write-Host "- openapi.json"
