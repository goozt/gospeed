.PHONY: build test vet lint clean docker release-snapshot

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
           -X github.com/goozt/gospeed/internal/version.Version=$(VERSION) \
           -X github.com/goozt/gospeed/internal/version.Commit=$(COMMIT) \
           -X github.com/goozt/gospeed/internal/version.Date=$(DATE)

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/gospeed ./cmd/gospeed
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/gospeed-server ./cmd/gospeed-server

test:
	go test -race -timeout 120s ./...

vet:
	go vet ./...

lint: vet

clean:
	rm -rf bin/ dist/

docker:
	docker build -t gospeed-server .

release-snapshot:
	goreleaser release --snapshot --clean
