TMP_DIR     ?= ./tmp
BIN_DIR     ?= ./bin

GOEXE       := $(shell go env GOEXE)
BUILD_FLAGS ?=

DOCKER_COMPOSE    ?= docker compose
APP_SERVICE       := server
DB_SERVICE        := db
MIGRATE_SERVICE   := migrate

WAIT_DB_READY     := sh scripts/wait-db-ready.sh
MIGRATE           := sh scripts/migrate.sh
MERGE_CODE        := sh scripts/merge-code.sh

SWAG_SOURCE_DIRS  := ./cmd/$(APP_SERVICE) ./internal/model ./internal/api
SWAG_SOURCES      := $(filter-out %_test.go,$(wildcard $(addsuffix /*.go,$(SWAG_SOURCE_DIRS))))
SWAG_DEST_DIR     := ./pkg/api/docs

empty   :=
space   := $(empty) $(empty)
comma   := ,
bold    := \033[1m
cmd     := \033[38;5;228m
var     := \033[1;36m
target  := \033[38;5;178m
comment := \033[38;5;65m
esc     := \033[0m


# source and dest for merge, patch, etc...
SRC   ?= .
DST   ?= 1

.PHONY: all

all: help

# ============================================
# RUN
# ============================================

.PHONY: run docker-run

run: generate test build db-up migrate-up ## run server (local)
	$(BIN_DIR)/server

docker-run: docker-build docker-up ## run server in docker
	$(DOCKER_COMPOSE) logs -f $(APP_SERVICE) $(MIGRATE_SERVICE)

# ============================================
# DOCKER COMMANDS
# ============================================

.PHONY: docker-build docker-up docker-down docker-down-volumes docker-logs
.PHONY: docker-db-up docker-db-down docker-db-down-volumes docker-db-shell

docker-build: ## build docker images
	$(DOCKER_COMPOSE) build

docker-up: ## start all services in docker
	$(DOCKER_COMPOSE) up -d $(APP_SERVICE) 
	@echo "Waiting for services to be ready..."
	@sleep 1
	$(DOCKER_COMPOSE) ps

docker-down: ## stop all docker services
	$(DOCKER_COMPOSE) down

docker-down-volumes: ## stop docker and remove volumes
	$(DOCKER_COMPOSE) down -v

docker-logs: ## show docker logs
	$(DOCKER_COMPOSE) logs -f

docker-db-up:
	$(DOCKER_COMPOSE) up -d $(DB_SERVICE)

docker-db-down: ## stop database container
	$(DOCKER_COMPOSE) down $(DB_SERVICE)

docker-db-down-volumes: ## stop database and remove volumes
	$(DOCKER_COMPOSE) down -v $(DB_SERVICE)

docker-db-shell: ## open psql in db container
	$(DOCKER_COMPOSE) exec $(DB_SERVICE) psql -U $${DB_USER:-postgres} -d $${DB_NAME:-postgres}

# ============================================
# DEVELOPMENT COMMANDS (local)
# ============================================

.PHONY: deps check-goose check-swag check-stringer check-tools build swag-generate go-generate generate test clean merge

deps: ## update deps
	go mod tidy

check-goose: ## install goose if need
	@which goose 2>/dev/null || go install github.com/pressly/goose/v3/cmd/goose@v3.27.1

check-swag: ## install swag if need
	@which swag 2>/dev/null || go install github.com/swaggo/swag/cmd/swag@v1.16.4

check-stringer: ## install stringer if need
	@which stringer 2>/dev/null || go install golang.org/x/tools/cmd/stringer@latest

check-tools: check-goose check-swag check-stringer


# Находим все поддиректории в cmd, которые потенциально могут быть бинарниками
CMDS := $(wildcard cmd/*)

# Генерируем список целей для бинарников
BINARIES := $(patsubst cmd/%,$(BIN_DIR)/%,$(CMDS))

# Шаблонное правило для сборки любого бинарника
$(BIN_DIR)/%: FORCE
	@mkdir -p $(@D)
	go build $(BUILD_FLAGS) -o $@$(GOEXE) ./cmd/$(notdir $@)

build: $(BINARIES) ## build all binaries

swag-generate: .swag-generate.done ## generate Swagger docs
	
.swag-generate.done: $(SWAG_SOURCES)
	swag fmt  -d $(subst $(space),$(comma),$(SWAG_SOURCE_DIRS))
	swag init -d $(subst $(space),$(comma),$(SWAG_SOURCE_DIRS)) -o $(SWAG_DEST_DIR)
	@touch $@

go-generate:
	go generate ./...

generate: go-generate swag-generate ## generate all

test: ## test
	go test ./internal/...

test-integration: ## test on real database
	go test ./tests/...

clean: ## remove temporary and binary files
	-rm -rf $(BIN_DIR) $(TMP_DIR) .*-generate.done

# ============================================
# DATABASE (local)
# ============================================

.PHONY: db-up db-down db-down-volumes db-shell

db-up: ## start database container for local dev
	@if [ -z "$$($(DOCKER_COMPOSE) ps -q $(DB_SERVICE))" ]; then \
		$(DOCKER_COMPOSE) up -d $(DB_SERVICE) && \
		DOCKER_COMPOSE='$(DOCKER_COMPOSE)' $(WAIT_DB_READY); \
	else \
		echo "Database already running"; \
	fi

db-down: docker-db-down ## alias for docker-db-down

db-down-volumes: docker-db-down-volumes ## alias for docker-db-down-volumes

db-shell: docker-db-shell ## alias for docker-db-shell

# ============================================
# MIGRATIONS (local)
# ============================================

.PHONY: migrate-up migrate-down migrate-status

migrate-up: ## apply all migrations (local)
	$(MIGRATE) up

migrate-down: ## rollback last migration (local)
	$(MIGRATE) down 1

migrate-status: ## show migration status (local)
	$(MIGRATE) status

# ============================================
# UTILS
# ============================================

.PHONY: FORCE help

FORCE:

merge: ## merge code to file for AI review
	@mkdir -p $(TMP_DIR)
	$(MERGE_CODE) $(SRC) > $(TMP_DIR)/$(DST).code 


patch: deps generate test ## make precommit patch
	@mkdir -p $(TMP_DIR)
	
	@(set -e; \
	staged_list="$(TMP_DIR)/staged_list.$$$$"; \
	unstaged_list="$(TMP_DIR)/unstaged_list.$$$$"; \
	git diff --staged --name-only -- $(SRC) > "$$staged_list"; \
	git diff --name-only -- $(SRC) > "$$unstaged_list"; \
	intersection=$$(grep -Fxf "$$staged_list" "$$unstaged_list" || true); \
	rm -f "$$staged_list" "$$unstaged_list"; \
	if [ -n "$$intersection" ]; then \
		echo "" >&2; \
		echo "$(bold)WARNING:$(esc) the following files have changes not staged for commit:" >&2; \
		echo "  (use \"git add <file>...\" to update what will be committed)" >&2; \
		printf '%s\n' $$intersection | sed 's/^/        /' >&2; \
		echo "" >&2; \
	fi)
	
	git diff --staged -- $(SRC) > $(TMP_DIR)/$(DST).patch
	@echo "Patch saved to $(TMP_DIR)/$(DST).patch"

help: ## show this help
	@printf "$(bold)Usage:$(esc)\n"
	@printf "  $(cmd)make$(esc) [$(var)VARIABLE$(esc)=value ...] [$(target)target$(esc) ...]\n"
	@printf "\n$(bold)Variables:$(esc)\n"
	@awk 'BEGIN {comment=""} \
		/^[a-zA-Z0-9_-]+[[:space:]]*\?=/ { \
			split($$0, a, /\?=/); \
			gsub(/^[ \t]+|[ \t]+$$/, "", a[1]); \
			gsub(/^[ \t]+|[ \t]+$$/, "", a[2]); \
			if ( prev ~ /^#/ ) { \
				gsub(/^[ \t]+|[ \t]+$$/, "", prev); \
				printf "  $(var)%-14s$(esc) = %-14s $(comment)%s$(esc)\n", a[1], a[2], prev; \
			} else { \
				printf "  $(var)%-14s$(esc) = %-14s\n", a[1], a[2]; \
			} \
		} \
		{ prev=$$0 }' \
		$(MAKEFILE_LIST)
	@printf "\n$(bold)Targets:$(esc)\n"
	@awk 'BEGIN {FS = ":.*?## "} \
		/^[a-zA-Z0-9_-]+:.*?## / \
		{printf "  $(target)%-22s$(esc) - %s\n", $$1, $$2}' \
		$(MAKEFILE_LIST)
	@printf "\n$(bold)Examples:$(esc)\n"
	@printf "  $(cmd)make$(esc) $(target)test$(esc)                     $(comment)# run tests$(esc)\n"
	@printf "  $(cmd)make$(esc) $(target)test-integration$(esc)         $(comment)# test service on real database$(esc)\n"
	@printf "  $(cmd)make$(esc) $(target)run$(esc)                      $(comment)# run server locally$(esc)\n"
	@printf "  $(cmd)make$(esc) $(target)docker-run$(esc)               $(comment)# run server in docker$(esc)\n"
	@printf "  $(cmd)make$(esc) $(target)docker-build docker-run$(esc)  $(comment)# rebuild docker images and restart$(esc)\n"
