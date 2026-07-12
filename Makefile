GOPATH=$(shell go env GOPATH)

.PHONY: test test-unit test-integration

test: test-integration

test-unit:
	@echo "start unit tests"
	go test -race -coverprofile=coverage.unit.txt ./...

test-integration: deps-reset
	@echo "start integration tests"
	go test -race -tags=integration -coverprofile=coverage.txt ./...

.PHONY: deps-up deps-down deps-reset deps-ps deps-logs deps-wait clean-docker-compose clean-test

deps-up:
	@echo "start redis and mongo"
	docker compose up -d redis mongo

deps-wait:
	@echo "wait for redis and mongo"
	docker compose up -d --wait redis mongo

deps-reset:
	@echo "reset redis and mongo"
	docker compose down -v || true
	docker compose up -d --wait redis mongo

deps-down:
	@echo "stop docker compose services"
	docker compose down

deps-ps:
	docker compose ps

deps-logs:
	docker compose logs -f redis mongo

clean-docker-compose: deps-down

clean-test: deps-down

delay:
	@echo "Run delay example with two terminals: make delay-consumer and make delay-publisher"

delay-consumer:
	@echo "start delay consumer"
	@cd examples/delay && go run -race ./consumer/main.go

delay-publisher:
	@echo "start delay publisher"
	@cd examples/delay && go run -race ./publisher/main.go

normal:
	@echo "Run normal example with two terminals: make normal-consumer and make normal-publisher"

normal-consumer:
	@echo "start normal consumer"
	@cd examples/normal && go run -race ./consumer/main.go

normal-publisher:
	@echo "start normal publisher"
	@cd examples/normal && go run -race ./publisher/main.go

sequential:
	@echo "Run sequential example with two terminals: make sequential-consumer and make sequential-publisher"

sequential-consumer:
	@echo "start sequential consumer"
	@cd examples/sequential && go run -race ./consumer/main.go

sequential-consumer-dlv:
	@echo "start sequential consumer"
	@cd examples/sequential && \
	go build -race -o server ./consumer/main.go && \
	dlv --headless --listen=:8888 --api-version=2 exec ./server

sequential-publisher:
	@echo "start sequential publisher"
	@cd examples/sequential && go run -race ./publisher/main.go

sequential-publisher-ack:
	@echo "start sequential publisher with ack"
	@cd examples/sequential && go run -race ./publisher-with-ack/main.go

ui:
	@echo "start ui on port:9090"
	@cd examples/ui && go run -race ./main.go

clean: deps-down
	@echo "done!"

GOLANGCI_LINT_VERSION=v2.12.2
GOLANGCI_LINT_TOOL = $(GOPATH)/bin/golangci-lint

lint: ## run all the lint tools, install golangci-lint if not exist,will use .golangci.yml config
	@if [ ! -x "$(GOLANGCI_LINT_TOOL)" ]; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) > /dev/null; \
	fi
	@$(if $(wildcard $(GOLANGCI_LINT_TOOL)),echo "Running golangci-lint...";) \
	$(GOLANGCI_LINT_TOOL)  run -v

FIELDALIGNMENT_TOOL = $(GOPATH)/bin/fieldalignment

vet: ## Field Alignment
	@if [ ! -x "$(FIELDALIGNMENT_TOOL)" ]; then \
		echo "Installing fieldalignment..."; \
		go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest; \
	fi
	@$(if $(wildcard $(FIELDALIGNMENT_TOOL)),echo "Running go vet with fieldalignment...";) \
	go vet -vettool=$(FIELDALIGNMENT_TOOL) ./... || exit 0

vet-fix: ##If fixed, the annotation for struct fields will be removed
	@if [ ! -x "$(FIELDALIGNMENT_TOOL)" ]; then \
		echo "Installing fieldalignment..."; \
		go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest; \
	fi
	@$(if $(wildcard $(FIELDALIGNMENT_TOOL)),echo "Running fieldalignment -fix...";) \
	$(FIELDALIGNMENT_TOOL) -fix ./... || exit 0

.PHONY: delay delay-consumer delay-publisher normal normal-consumer normal-publisher \
 		sequential sequential-publisher sequential-consumer sequential-consumer-dlv sequential-publisher-ack ui clean \
		lint vet vet-fix
