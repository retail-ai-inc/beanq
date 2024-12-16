GOPATH=$(shell go env GOPATH)

delay: delay-publisher delay-consumer

delay-consumer:
	@echo "start delay consumer"
	@cd examples/delay/consumer && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

delay-publisher:
	@echo "start delay publisher"
	@cd examples/delay/publisher && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

normal: normal-publisher normal-consumer

normal-consumer:
	@echo "start normal consumer"
	@cd examples/normal/consumer && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

normal-publisher:
	@echo "start normal publisher"
	@cd examples/normal/publisher && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

sequential: sequential-publisher sequential-consumer

sequential-consumer:
	@echo "start sequential consumer"
	@cd examples/sequential/consumer && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

dynamic-consumer:
	@echo "start dynamic sequential consumer"
	@cd examples/sequential/consumer-dynamic && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

sequential-publisher:
	@echo "start sequential publisher"
	@cd examples/sequential/publisher && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

sequential-publisher-ack:
	@echo "start sequential publisher with ack"
	@cd examples/sequential/publisher-with-ack && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

dynamic-publisher:
	@echo "start dynamic sequential publisher with ack"
	@cd examples/sequential/publisher-dynamic && \
	sed -i 's/"host": "localhost"/"host": "redis-beanq"/' ./env.json && \
	go run -race ./main.go

clean:
	@echo "start.."

	@echo "delay clean"
	@cd examples/delay/consumer && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json
	@cd examples/delay/publisher && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json

	@echo "normal clean"
	@cd examples/normal/consumer && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json
	@cd examples/normal/publisher && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json

	@echo "sequential clean"
	@cd examples/sequential/consumer && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json
	@cd examples/sequential/publisher && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json
	@cd examples/sequential/publisher-with-ack && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json
	@cd examples/sequential/publisher-dynamic && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json
	@cd examples/sequential/consumer-dynamic && \
	sed -i 's/"host": "redis-beanq"/"host": "localhost"/' ./env.json

	@echo "done!"

GOLANGCI_LINT_VERSION=v1.55.2
GOLANGCI_LINT_TOOL = $(GOPATH)/bin/golangci-lint
lint: ## run all the lint tools, install golangci-lint if not exist
	@if [ ! -x "$(GOLANGCI_LINT_TOOL)" ]; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) > /dev/null; \
	fi
	@$(if $(wildcard $(GOLANGCI_LINT_TOOL)),echo "Running golangci-lint...";) \
	$(GOLANGCI_LINT_TOOL) --verbose run

FIELDALIGNMENT_TOOL = $(GOPATH)/bin/fieldalignment
.PHONY: vet
vet: ## Field Alignment
	@if [ ! -x "$(FIELDALIGNMENT_TOOL)" ]; then \
		echo "Installing fieldalignment..."; \
		go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest; \
	fi
	@$(if $(wildcard $(FIELDALIGNMENT_TOOL)),echo "Running go vet with fieldalignment...";) \
	go vet -vettool=$(FIELDALIGNMENT_TOOL) ./... || exit 0

.PHONY: vet-fix
vet-fix: ##If fixed, the annotation for struct fields will be removed
	@if [ ! -x "$(FIELDALIGNMENT_TOOL)" ]; then \
		echo "Installing fieldalignment..."; \
		go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest; \
	fi
	@$(if $(wildcard $(FIELDALIGNMENT_TOOL)),echo "Running fieldalignment -fix...";) \
	$(GOPATH)/bin/fieldalignment -fix ./... || exit 0


.PHONY: delay delay-consumer delay-publisher normal normal-consumer normal-publisher\
 		sequential sequential-publisher sequential-consumer sequential-publisher-ack clean
