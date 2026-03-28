.PHONY: build build-agents run-platform run-gateway run-runner run-agent deploy deploy-web deploy-api stop restart status logs reset

deploy:
	@./deploy.sh start

deploy-web:
	@./deploy.sh web

deploy-api:
	@./deploy.sh api

stop:
	@./deploy.sh stop

restart:
	@./deploy.sh restart

status:
	@./deploy.sh status

logs:
	@./deploy.sh logs

reset:
	@./deploy.sh reset

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

build-agents:
	@echo "Cross-compiling enx-agent for all platforms..."
	@mkdir -p bin/agents
	@for os in linux windows darwin; do \
		for arch in amd64 arm64; do \
			ext=""; \
			if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
			name="enx-agent-$$os-$$arch$$ext"; \
			echo "  Building $$name..."; \
			cd apps/agent-core && CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
				go build -ldflags="-s -w" -o ../../bin/agents/$$name ./cmd/enx-agent && cd ../..; \
			echo "  ✓ $$name"; \
		done; \
	done
	@echo "All agent binaries built in ./bin/agents/"

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
	cd apps/console-web && pnpm run dev
