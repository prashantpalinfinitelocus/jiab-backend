.PHONY: build dev clean up down logs docs rebuild rebuild-no-cache local prod

APP_NAME := coca_cola_backend

# Build the Go application
build:
	go build -o main .

# Run the Go application
dev:
	go run main.go server

# Clean the generated files
clean:
	go clean
	rm -f main

# Run the Docker container
up:
	docker compose up -d

# Stop and remove Docker containers
down:
	docker compose down

# Follow Docker container logs
logs:
	docker compose logs -f --tail=100

# Generate API documentation
docs:
	swag init -g cmd/server.go

# Rebuild and restart Docker containers
rebuild:
	docker compose build
	docker compose up -d

# Rebuild (no cache) and restart Docker containers
rebuild-no-cache:
	docker compose build --no-cache
	docker compose up -d

up-ssh:
	docker compose -f docker-compose.yml -f docker-compose.ssh.yml up -d

rebuild-ssh:
	docker compose -f docker-compose.yml -f docker-compose.ssh.yml build 
	docker compose -f docker-compose.yml -f docker-compose.ssh.yml up -d

rebuild-ssh-no-cache:
	docker compose -f docker-compose.yml -f docker-compose.ssh.yml build --no-cache
	docker compose -f docker-compose.yml -f docker-compose.ssh.yml up -d

build-kafka:
	docker compose -f docker-compose.kafka.yml build

up-kafka:
	docker compose -f docker-compose.kafka.yml up -d

down-kafka:
	docker compose -f docker-compose.kafka.yml down
