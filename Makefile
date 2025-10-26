# Makefile for biomuzak local dev (macOS/bash)
# Loads variables from .env when present and exports them to shell commands

SHELL := /bin/bash

DOCKER_COMPOSE := docker compose

# Aggregate service list for logs convenience
SERVICES := postgres backend frontend audio-processor

.PHONY: help dev up build down restart logs tail backend-logs frontend-logs audio-logs db-logs open url psql db-psql db-shell status clean prune

help:
	@echo "biomuzak dev helper targets:"
	@echo "  make dev         # build and start all services in background"
	@echo "  make up          # alias of dev"
	@echo "  make down        # stop and remove containers (keeps volumes)"
	@echo "  make clean       # stop and remove containers + volumes"
	@echo "  make restart     # restart the stack"
	@echo "  make logs        # follow logs for all services"
	@echo "  make backend-logs|frontend-logs|audio-logs|db-logs"
	@echo "  make open        # open the web app in your browser"
	@echo "  make url         # print the app URL"
	@echo "  make psql        # open psql inside the postgres container"
	@echo "  make status      # show docker compose status"

# Build and start full dev stack
dev: build
	$(DOCKER_COMPOSE) up -d
	@$(MAKE) url

# Alias
up: dev

# Build images
build:
	$(DOCKER_COMPOSE) build --pull

# Stop and remove containers (keep volumes)
down:
	$(DOCKER_COMPOSE) down

# Stop, remove and nuke volumes (DB/uploads)
clean:
	$(DOCKER_COMPOSE) down -v

# Restart the stack
restart:
	$(MAKE) down
	$(MAKE) dev

# Follow logs for all services
logs:
	$(DOCKER_COMPOSE) logs -f $(SERVICES)

backend-logs:
	$(DOCKER_COMPOSE) logs -f backend

frontend-logs:
	$(DOCKER_COMPOSE) logs -f frontend

audio-logs:
	$(DOCKER_COMPOSE) logs -f audio-processor

db-logs:
	$(DOCKER_COMPOSE) logs -f postgres

# Open the app in the default browser (macOS)
open: url
	@open "http://localhost:$(FRONTEND_PORT)"

# Print the app URL
url:
	@echo "App: http://localhost:$(FRONTEND_PORT)"
	@echo "API: http://localhost:$(BACKEND_PORT)"

# Open a psql shell inside the Postgres container
# Relies on docker-compose container_name: music_db
psql db-psql:
	@docker exec -it music_db sh -lc 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

# Optional: connect using host-installed psql if available
# Uses discrete parameters to avoid URL-encoding issues
db-shell:
	@echo "Tip: If this fails due to env parsing, use: make psql"

status:
	$(DOCKER_COMPOSE) ps

# Housekeeping: remove dangling images
prune:
	@docker image prune -f
