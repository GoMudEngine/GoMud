

.DEFAULT_GOAL := build

VERSION ?= $(shell git rev-parse HEAD)
BIN ?= go-mud-server
DOCKER_COMPOSE := docker-compose -f compose.yml

export GOFLAGS := -mod=mod

## Build Targets

.PHONY: docker_build 
docker_build: 
	TAG=$(VERSION) $(DOCKER_COMPOSE) build server

DOCKER_CMD ?= bash

.PHONY: console
console:
	@docker run --rm -it --init \
			-v "$(PWD)":/src \
			-w /src \
			golang:1.21.3-alpine3.18 \
			$(DOCKER_CMD)

docker-%:
	@$(MAKE) console DOCKER_CMD="make $(patsubst docker-%,%,$@)"

#
#
# For a complete list of GOOS/GOARCH combinations:
# Run: go tool dist list
#
#

.PHONY: build_rpi_zero2w
build_rpi_zero2w: generate ### Build a binary for a raspberry pi zero 2w
	env GOOS=linux GOARCH=arm64 go build -o $(BIN)-rpi

.PHONY: build_win64
build_win64: generate ### Build a binary for 64bit windows
	env GOOS=windows GOARCH=amd64 go build -o $(BIN)-win64.exe

.PHONY: build_linux64
build_linux64: generate ### Build a binary for linux
	env GOOS=linux GOARCH=amd64 go build -o $(BIN)-linux64

.PHONY: build
build: validate build_local  ### Validate the code and build the binary.

.PHONY: build_local
build_local: generate
	CGO_ENABLED=0 go build -trimpath -a -o $(BIN)

.PHONY: generate
generate: ### Generates include directives for modules
	@go generate


# Clean both development and production containers
.PHONY: clean
clean:
	$(DOCKER_COMPOSE) down --volumes --remove-orphans
	docker system prune -a

.PHONY: clean-instances
clean-instances: ### Deletes all room instance data. Starts the world fresh.
	rm -Rf _datafiles/world/default/rooms.instances
	rm -Rf _datafiles/world/empty/rooms.instances

## Run Targets

.PHONY: run 
run: generate  ### Build and run server.
	@go run .

.PHONY: run-new
run-new: clean-instances generate run ### Deletes instance data and runs server

.PHONY: run-docker
run-docker: ### Build and run server in docker.
	$(DOCKER_COMPOSE) up --build --remove-orphans server


.PHONY: client
client: ### Build and run client terminal client
	$(DOCKER_COMPOSE) run --rm terminal telnet go-mud-server 33333

.PHONY: image_tag
image_tag:
	@echo $(VERSION)

.PHONY: port
port:
	@$(eval PORT := $(shell $(DOCKER_COMPOSE) port server 8080))
	@echo $(PORT)

.PHONY: shell
shell:
	@$(eval CONTAINER_NAME := $(shell docker ps --filter="name=mud" --format '{{.Names}}' ))
	docker exec -it $(CONTAINER_NAME) /bin/sh 

#
#
# Local code run/test
#
#
.PHONY: validate
validate: fmtcheck vet

.PHONY: test
test: js-lint
	@go test -timeout 300s -race ./...

.PHONY: test-regression
test-regression: ### Run regression test suite
	@go test -run "TestRegression_" -v ./...

.PHONY: test-smoke
test-smoke: ### Run smoke test suite
	@go test -run "TestSmoke_" -v ./...

# Automates the Pre-Push SOP step "boot the server locally and confirm it starts
# cleanly past data-file loading". Loads every YAML data file through the real
# boot path and fails on malformed content. Opt-in via env var because it loads
# the whole world (~20s) and populates every package global.
.PHONY: boot-check
boot-check: ### Verify every data file loads (replaces the manual pre-push boot)
	@DOGMUD_BOOT_SMOKE=1 go test -run "TestSmoke_ServerBootsCleanWithRealData|TestSmoke_AllDialogueFilesParse" -v -timeout 300s .

.PHONY: coverage
coverage: 
	@mkdir -p bin/covdatafiles && \
	go test ./... -coverprofile=bin/covdatafiles/cover.out && \
	go tool cover -html=bin/covdatafiles/cover.out && \
	rm -rf bin

.PHONY: js-lint
js-lint:  ### Run Javascript linter
#   Grep filtering it to remove errors reported by docker image around npm packages
#   if "### errors" is found in the output, exits with an error code of 1
#   This should allow us to use it in CI/CD
	@docker run --rm -v "$(PWD)":/app -w /app node:20 npx jshint . \
	 2>&1 | grep -v "^npm " | tee /dev/stderr | grep -Eq "^[0-9]+ errors" && exit 1 || true

#
#
# Cert generation for testing
#
#
.PHONY: cert-clean
cert-clean:
	rm -f server.crt server.key

.PHONY: cert
cert: server.crt server.key

# This rule generates both files in one go using OpenSSL.
server.crt server.key:
	openssl req -x509 -nodes -newkey rsa:4096 \
		-keyout server.key -out server.crt \
		-days 365 -subj "/CN=localhost"



# Go targets

.PHONY: fmt
fmt:
	@go fmt ./...

# Reports unformatted files without rewriting them, matching the CI gate in
# .github/actions/codegen-and-test. The previous implementation used
# `go fmt ./...`, which silently REWROTE the tree and then failed — so a
# "check" mutated your working copy. Use `make fmt` to actually apply changes.
.PHONY: fmtcheck
fmtcheck:
	@set -e; \
	unformatted=$$(git ls-files '*.go' | grep -v '^vendor/' | xargs -r gofmt -l); \
	if [ ! -z "$$unformatted" ]; then \
		echo "These files are not gofmt-clean. Fix with 'make fmt':"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: mod
mod:
	@go mod vendor
	@go mod tidy
	@go mod verify


.PHONY: vet
vet:
	@go vet ./...

.PHONY: set_gopath
set_gopath:
ifeq ($(OS),Windows_NT)
	setx PATH "$(PATH);mytest" -m
else
	export GOPATH=$GOPATH:$(pwd)
endif

.PHONY: view_pprof_mem
view_pprof_mem:
	go tool pprof -http=:8989 source/_datafiles/profiles/mem.pprof

#
# Help target - greps and formats special comments to form a "help" command for makefiles
#
## Help
.PHONY: help
help:                 ### List makefile targets.
# 1. grep for any lines starting with "##" or containing "\s###\s"
# 2. Align targets/comments with string padding
# 3. Wrap lines starting with "##" in ANSI escape codes (color) as headers
# 4. Wrap lines containing "\s###\s" in ANSI escape codes (color) as commands
# 5. Add new lines before any that aren't prefixed with space (Headers)
	@grep -hE "^##\s|\s###\s" $(MAKEFILE_LIST) \
		| awk -F'[[:space:]]###[[:space:]]' '{printf "%-36s### %s\n", substr($$1,1,35), $$2}' \
		| sed -E "s/^## ([^#]*)#*/`printf "\033[90;3m"`\1`printf "\033[0m"`/" \
		| sed "s/\(.*\):\(.*\)###\(.*\)$$/  `printf "\033[93m"`\1:`printf "\033[36m"`\2`printf "\033[97m"`-\3`printf "\033[0m"`/" \
    	| sed "/^[^[:blank:]]/{x;p;x;}"
	@printf "\n"

