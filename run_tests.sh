#!/bin/bash
set -e

# First run the database container
echo "Starting test database container..."
docker compose -f docker-compose.test.yml up -d mongo-test

echo "Waiting for database to be ready..."
# The test service has a depends_on with service_healthy for postgres,
# so docker compose will automatically wait before starting tests.

# Then the backend tests
echo "Running backend tests inside container..."
docker compose -f docker-compose.test.yml up -d test-runner

# Wait for the test container to finish running
docker compose -f docker-compose.test.yml wait test-runner

# After that show the logs for the tests
echo "================ TEST LOGS ================"
docker compose -f docker-compose.test.yml logs test-runner
echo "==========================================="

# And then delete all the containers related to the tests
echo "Deleting test containers and volumes..."
docker compose -f docker-compose.test.yml down -v

echo "Test execution finished safely."
