GITCOMMIT?=$(shell git rev-parse HEAD)
GITDESCRIBE?=$(shell git describe --tags 2>/dev/null || echo nightly)
BUILD_EPOCH?=$(shell date +%s)

# Define platforms and architectures
PLATFORMS := linux darwin windows
COMMON_ARCHITECTURES := 386 amd64 arm
DARWIN_ARCHITECTURES := amd64 arm64 # darwin/386 is obsolete, arm is arm64

all: clean build-all package-all

build-all:
	@echo "Building binaries..."
	@for GOOS in $(PLATFORMS); do \
		CURRENT_ARCHS=""; \
		if [ "$$GOOS" = "darwin" ]; then \
			CURRENT_ARCHS="$(DARWIN_ARCHITECTURES)"; \
		else \
			CURRENT_ARCHS="$(COMMON_ARCHITECTURES)"; \
		fi; \
		for GOARCH in $$CURRENT_ARCHS; do \
			echo "Building for $$GOOS/$$GOARCH..."; \
			OUT_DIR="bin/$$GOOS-$$GOARCH"; \
			mkdir -p $$OUT_DIR; \
			EXE_NAME="igmpqd"; \
			if [ "$$GOOS" = "windows" ]; then \
				EXE_NAME="igmpqd.exe"; \
			fi; \
			GOOS=$$GOOS GOARCH=$$GOARCH go build -v -o "$$OUT_DIR/$$EXE_NAME" \
				-ldflags="-X main.GitCommit=$(GITCOMMIT) -X main.GitDescribe=$(GITDESCRIBE) -X main.BuildTime=$(BUILD_EPOCH)" .; \
		done; \
	done

package-all:
	@echo "Packaging binaries..."
	/bin/bash ./build-release-bins.sh

clean:
	@echo "Cleaning up old binaries..."
	@rm -rf bin/*
	@rm -rf dist/*
