SHELL := /bin/zsh
.SHELLFLAGS := -eu -o pipefail -c

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BACKEND_LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

.PHONY: backend-build backend-test backend-run desktop-build desktop-dev desktop-test web-build web-dev web-test ci deps migrate-up migrate-down migrate-create

backend-build:
	cd backend && mkdir -p bin && GOCACHE=/tmp/go-build go build -ldflags "$(BACKEND_LDFLAGS)" -o bin/sukumad ./cmd/api

backend-test:
	cd backend && GOCACHE=/tmp/go-build go test ./...

backend-run:
	cd backend && GOCACHE=/tmp/go-build go run ./cmd/api

desktop-build:
	cd desktop/frontend && npm run build

desktop-dev:
	cd desktop && GOROOT=/usr/local/go PATH=/usr/local/go/bin:$$PATH wails dev -compiler /usr/local/go/bin/go

desktop-test:
	cd desktop/frontend && npm test

web-build:
	cd web && npm run build

web-dev:
	cd web && npm run dev

web-test:
	cd web && npm test

ci: backend-test desktop-test desktop-build

migrate-up:
	cd backend && GOCACHE=/tmp/go-build go run ./cmd/migrate up

migrate-down:
	cd backend && GOCACHE=/tmp/go-build go run ./cmd/migrate down

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "usage: make migrate-create name=<migration_name>"; \
		exit 1; \
	fi
	cd backend && GOCACHE=/tmp/go-build go run ./cmd/migrate create -name $(name)

deps:
	cd backend && GOCACHE=/tmp/go-build go mod tidy
	cd desktop/frontend && npm install
