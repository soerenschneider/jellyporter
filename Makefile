BUILD_DIR = builds
BINARY_NAME = jellyporter
MODULE = github.com/soerenschneider/$(BINARY_NAME)
CHECKSUM_FILE = checksum.sha256
SIGNATURE_KEYFILE = ~/.signify/github.sec
DOCKER_PREFIX = ghcr.io/soerenschneider

generate:
	go generate  ./...

tests:
	go test ./... -race -covermode=atomic -coverprofile=coverage.out
	go tool cover -html=coverage.out -o=coverage.html
	go tool cover -func=coverage.out -o=coverage.out

clean:
	git diff --quiet || { echo 'Dirty work tree' ; false; }
	rm -rf ./$(BUILD_DIR)

build: generate version-info
	CGO_ENABLED=1 go build -ldflags="-w -X '$(MODULE)/main.BuildVersion=${VERSION}' -X '$(MODULE)/main.CommitHash=${COMMIT_HASH}'" -o $(BINARY_NAME) .

release: clean version-info cross-build
	cd $(BUILD_DIR) && sha256sum * > $(CHECKSUM_FILE) && cd -

signed-release: release
	pass keys/signify/github | signify -S -s $(SIGNATURE_KEYFILE) -m $(BUILD_DIR)/$(CHECKSUM_FILE)
	gh-upload-assets -o soerenschneider -r $(BINARY_NAME) -f ~/.gh-token builds

cross-build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1       go build -ldflags="-w -X '$(MODULE)/main.BuildVersion=${VERSION}' -X '$(MODULE)/main.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64     .
	GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=1 go build -ldflags="-w -X '$(MODULE)/main.BuildVersion=${VERSION}' -X '$(MODULE)/main.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv6     .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1       go build -ldflags="-w -X '$(MODULE)/main.BuildVersion=${VERSION}' -X '$(MODULE)/main.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-aarch64   .

docker-build:
	docker build -t "$(DOCKER_PREFIX)/$(BINARY_NAME)" .

version-info:
	$(eval VERSION := $(shell git describe --tags --abbrev=0 || echo "dev"))
	$(eval COMMIT_HASH := $(shell git rev-parse HEAD))

fmt:
	find . -iname "*.go" -exec go fmt {} \; 

lint:
	golangci-lint run

pre-commit-init:
	pre-commit install
	pre-commit install --hook-type commit-msg

pre-commit-update:
	pre-commit autoupdate
