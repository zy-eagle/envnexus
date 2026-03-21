.PHONY: build run-platform run-gateway run-runner run-agent

build:
	@echo "Building platform-api..."
	cd services/platform-api && go build -o ../../bin/platform-api ./cmd/platform-api
	@echo "Building session-gateway..."
	cd services/session-gateway && go build -o ../../bin/session-gateway ./cmd/session-gateway
	@echo "Building job-runner..."
	cd services/job-runner && go build -o ../../bin/job-runner ./cmd/job-runner
	@echo "Building agent-core..."
	cd apps/agent-core && go build -o ../../bin/enx-agent ./cmd/enx-agent
	@echo "Build complete. Binaries are in ./bin"

run-platform:
	cd services/platform-api && go run ./cmd/platform-api

run-gateway:
	cd services/session-gateway && go run ./cmd/session-gateway

run-runner:
	cd services/job-runner && go run ./cmd/job-runner

run-agent:
	cd apps/agent-core && go run ./cmd/enx-agent

run-desktop:
	cd apps/agent-desktop && npm start

run-web:
	cd apps/console-web && npm run dev
