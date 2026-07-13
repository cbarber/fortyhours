MODULE  := github.com/cbarber/fortyhours
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X '$(MODULE)/internal/cli.Version=$(VERSION)'
DIST    := dist

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o fortyhours ./cmd/fortyhours

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	go vet ./...
	gofmt -l .
	golangci-lint run ./...

.PHONY: generate
generate:
	go generate ./...

.PHONY: dist
dist: $(addprefix $(DIST)/fortyhours-,$(subst /,-,$(PLATFORMS)))

$(DIST)/fortyhours-%:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) \
		go build -ldflags "$(LDFLAGS)" -o $@ ./cmd/fortyhours

.PHONY: clean
clean:
	rm -rf $(DIST) fortyhours
