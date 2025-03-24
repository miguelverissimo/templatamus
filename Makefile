VERSION := $(shell git describe --tags --always --dirty)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

.PHONY: build
build:
	go build -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)" -o templatamus cmd/templatamus/main.go

.PHONY: install
install:
	go install -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)" ./cmd/templatamus

.PHONY: clean
clean:
	rm -f templatamus 