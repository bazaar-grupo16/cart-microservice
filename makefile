up:
	docker compose up -d
stop:
	docker compose down
logs:
	docker compose logs -f
ps:
	docker compose ps
build:
	docker compose build
clean:
	docker compose down -v
test:
	./run_tests.sh
set-environment:
	./set_env_variables.sh

postgres:
	docker compose up -d postgres

run:
	docker compose up --build

lint:
	golangci-lint run --timeout 5m