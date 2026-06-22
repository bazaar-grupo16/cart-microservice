<<<<<<< Updated upstream
swag init -g src/main.go -o src/docs --parseDependency --parseInternal
=======
#!/bin/bash
# Exit immediately if a command exits with a non-zero status
set -e

# Navigate to the root directory of the microservice (where go.mod is located)
cd "$(dirname "$0")/.."

echo "Generating OpenAPI documentation using Swaggo..."

# Run swag init. It parses the annotations in src/main.go and controllers.
# It outputs swagger.json and swagger.yaml to the docs/ directory.
go run github.com/swaggo/swag/cmd/swag@latest init -g src/main.go --output docs --outputTypes json,yaml

# Create the .agents directory if it doesn't exist
mkdir -p .agents

# Copy/rename the generated JSON spec to the requested location in .agents
cp docs/swagger.json "openapi.json"

echo "OpenAPI contract successfully generated:"
echo "- docs/swagger.json"
echo "- docs/swagger.yaml"
echo "- openapi.json"
>>>>>>> Stashed changes
