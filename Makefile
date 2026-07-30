GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: generate generate-contract generate-sql generate-check fmt fmt-check vet lint test test-integration test-race build clean

# PostgreSQL-backed tests are opt-in so ordinary unit tests remain database-independent.
test-integration:
	@test -n "$$DATABASE_URL" || (echo "DATABASE_URL is required for integration tests" >&2; exit 1)
	$(GO) test -tags=integration ./test/integration

generate: generate-contract generate-sql

generate-contract:
	cd contract/gen && $(GO) tool oapi-codegen --config oapi-codegen.yaml ../../api/openapi.yaml

generate-sql:
	@if test -f sqlc.yaml; then \
		$(GO) tool sqlc generate; \
	else \
		echo "sqlc generation skipped: sqlc.yaml not added yet"; \
	fi

generate-check: generate
	@if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then \
		git diff --exit-code -- contract/gen database/query; \
	else \
		echo "generation completed (clean-tree check skipped: no Git worktree)"; \
	fi

fmt:
	gofmt -w $$(find . -type f -name '*.go' -not -path './.cache/*')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -type f -name '*.go' -not -path './.cache/*'))"

vet:
	$(GO) vet ./...

lint:
	$(GOLANGCI_LINT) run ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

build:
	mkdir -p bin
	$(GO) build -trimpath -o bin/tidekeepers-api ./cmd/api

clean:
	rm -rf bin coverage.out
