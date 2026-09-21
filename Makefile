.PHONY: run run-refresh down build test integration-test fmt

run:
	docker compose up --build -d decision-service decision-gateway

run-refresh:
	docker compose --profile refresh up --build -d

down:
	docker compose --profile refresh down --remove-orphans

build:
	docker compose --profile refresh build

test:
	docker build --target test -t policy-snapshot-decision-test ./decision-service
	docker build --target test -t policy-snapshot-gateway-test ./decision-gateway
	docker build --target test -t policy-snapshot-registry-test ./policy-registry

integration-test:
	docker compose --profile refresh up --build -d --wait --wait-timeout 60
	python3 -m unittest discover -s integration-tests -p 'test_*.py'

fmt:
	docker run --rm -v "$(CURDIR)/decision-service:/src" -w /src golang:1.24-alpine gofmt -w .
	docker run --rm -v "$(CURDIR)/decision-gateway:/src" -w /src golang:1.24-alpine gofmt -w .
