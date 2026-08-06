.PHONY: build run test lint vet fmt tidy \
        web-install web-dev web-build web-lint \
        docker-build docker-up docker-down docker-logs \
        clean

BINARY := bin/netwatch-server

## Backend

build:
	go build -o $(BINARY) ./cmd/server

run: build
	./$(BINARY) --config ./configs

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

tidy:
	go mod tidy

lint: fmt vet

## Frontend

web-install:
	cd web && npm install

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

web-lint:
	cd web && npm run lint

## Docker

docker-build:
	docker compose build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

## Housekeeping

clean:
	rm -rf $(BINARY) web/.next data
