.PHONY: fmt fmt-check lint check test build start clean

fmt:
	deno fmt

fmt-check:
	deno task fmt:check

lint:
	deno task lint

check:
	deno task check

test:
	deno task test

build:
	deno check --unstable-kv --unstable-cron server/main.ts

start:
	deno task start

clean:
	rm -rf bin coverage.out
