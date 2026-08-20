# const
PROJECT_NAME := beerer-bot
ENV_PATH := ./build/.dev.env
DOCKER_COMPOSE_PATH := ./build/compose.yaml

# exec
GO := go
DOCKER_COMPOSE := docker compose -f $(DOCKER_COMPOSE_PATH) --env-file $(ENV_PATH) -p $(PROJECT_NAME)


# ======================================================================
# APP MANAGEMENT
# ======================================================================
.PHONY: build
build: clean build/bin/beerer-bot

.PHONY: build/bin/beerer-bot
build/bin/beerer-bot:
	$(GO) build -o build/bin/beerer-bot ./cmd/bot

.PHONY: run/beerer-bot
run/beerer-bot:
	./build/bin/beerer-bot

.PHONY: clean
clean:
	rm -rf ./build/bin

.PHONY: tidyvendor
tidyvendor:
	$(GO) mod tidy
	$(GO) mod vendor


# ======================================================================
# DOCKER-COMPOSE
# ======================================================================
.PHONY: docker/build
docker/build:
	$(DOCKER_COMPOSE) build

.PHONY: docker/up
docker/up:
	$(DOCKER_COMPOSE) up -d

.PHONY: docker/stop
docker/stop:
	$(DOCKER_COMPOSE) stop

.PHONY: docker/down
docker/down:
	$(DOCKER_COMPOSE) down -v
