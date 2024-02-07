NATS_SERVER=34.107.87.29

# Define the paths to the source code and build artifacts
SRC_PATH=.
BUILD_PATH=./build
DOCKER_PATH=./docker
DOCKERFILE_PROD=$(DOCKER_PATH)/Dockerfile
DOCKERFILE_DEV=$(DOCKER_PATH)/Dockerfile.dev

# Define the name of the binary and Docker image
BINARY_NAME=osmosis-publisher
DOCKER_IMAGE_NAME=osmosis-publisher

# Define the build flags for go build
BUILD_FLAGS=-ldflags="-s -w"

# Define the command for running tests
TEST_CMD=go test -race ./...

# Define the command for running the development version
DEV_CMD=go run github.com/itohio/xnotify@v0.3.1 -i .env -i internal -i pkg -i cmd --batch 100 --verbose --trigger -- make build-debug -- $(BUILD_PATH)/$(BINARY_NAME) --verbose $(ARGS)

CGO_ENABLED?=1

.PHONY: all
all: build

.PHONY: gen
gen:
	# Run go generate to generate any required files
	go generate ./...

.PHONY: build
build:
	# Build the production binary
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -o $(BUILD_PATH)/$(BINARY_NAME) $(SRC_PATH)
	ldd $(BUILD_PATH)/$(BINARY_NAME) | grep "libwasmvm.*.so" | awk '{print $$3}' | xargs -I '{}' cp '{}' $(BUILD_PATH)

.PHONY: build-debug
build-debug:
	# Build the debug binary
	go build -o $(BUILD_PATH)/$(BINARY_NAME) $(SRC_PATH)

.PHONY: test
test:
	$(TEST_CMD)

.PHONY: serve
serve:
	# Run the development version of the program
	$(DEV_CMD)

.PHONY: docker-build
docker-build:
	# Build the production Docker image
	docker build -f $(DOCKERFILE_PROD) -t $(DOCKER_IMAGE_NAME):latest .

.PHONY: docker-build-dev
docker-build-dev:
	# Build the development Docker image
	docker build -f $(DOCKERFILE_DEV) -t $(DOCKER_IMAGE_NAME):dev .

.PHONY: clean
clean:
	# Remove the build artifacts
	rm -rf $(BUILD_PATH)

.PHONY: all-tests
all-tests: test docker-build

.PHONY: monitor
monitor:
	nats --server $(NATS_SERVER) --creds ../nats.creds sub "syntropy.osmosis.>" | awk 'BEGIN { RS="\\n\\[#"; ORS="" } /Received on "syntropy.osmosis.telemetry"/ { block = 1; next } /^$/ { block = 0; next } !block'


.PHONY: help
help:
	@echo "Makefile for building, testing, and Dockerizing a Go application"
	@echo ""
	@echo "Usage:"
	@echo "  make gen             Generate necessary files"
	@echo "  make build           Build the production binary"
	@echo "  make build-debug     Build the debug binary"
	@echo "  make test            Run all tests"
	@echo "  make serve           Run the development version of the program"
	@echo "  make docker-build    Build the production Docker image"
	@echo "  make docker-build-dev Build the development Docker image"
	@echo "  make clean           Remove build artifacts"
	@echo "  make all-tests       Run all tests and build the production Docker image"
	@echo "  make help            Display this help message"
