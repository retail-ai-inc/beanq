GOPATH=$(shell go env GOPATH)

delay: delay-publisher delay-consumer

delay-consumer:
	@echo "start delay consumer"
	@cd examples/delay/consumer && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

delay-publisher:
	@echo "start delay publisher"
	@cd examples/delay/publisher && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

normal: normal-publisher normal-consumer

normal-consumer:
	@echo "start normal consumer"
	@cd examples/normal/consumer && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

normal-publisher:
	@echo "start normal publisher"
	@cd examples/normal/publisher && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

sequential: sequential-publisher sequential-consumer

sequential-consumer:
	@echo "start sequential consumer"
	@cd examples/sequential/consumer && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

sequential-consumer-dlv:
	@echo "start sequential consumer"
	@cd examples/sequential/consumer && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go build -race -o server ./main.go && \
	dlv --headless --listen=:8888 --api-version=2 exec ./server

sequential-publisher:
	@echo "start sequential publisher"
	@cd examples/sequential/publisher && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

sequential-publisher-ack:
	@echo "start sequential publisher with ack"
	@cd examples/sequential/publisher-with-ack && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

ui: ## Start ui
	@echo "start ui on port:9090"
	@cd examples/ui && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

clean: ## Clean up all test configuration information
	@echo "start.."

	@echo "delay clean"
	@cd examples/delay/consumer && \
	jq '.redis.host = "localhost" | .history.mongo.host = "localhost"' ./env.json > temp.json && \
	mv temp.json env.json
	@cd examples/delay/publisher && \
	jq '.redis.host = "localhost" | .history.mongo.host = "localhost"' ./env.json > temp.json && \
	mv temp.json env.json

	@echo "normal clean"
	@cd examples/normal/consumer && \
	jq '.redis.host = "localhost" | .history.mongo.host = "localhost"' ./env.json > temp.json && \
	mv temp.json env.json
	@cd examples/normal/publisher && \
	jq '.redis.host = "localhost" | .history.mongo.host = "localhost"' ./env.json > temp.json && \
	mv temp.json env.json

	@echo "sequential clean"
	@cd examples/sequential/consumer && \
	jq '.redis.host = "localhost" | .history.mongo.host = "localhost"' ./env.json > temp.json && \
	mv temp.json env.json
	@cd examples/sequential/publisher && \
	jq '.redis.host = "localhost" | .history.mongo.host = "localhost"' ./env.json > temp.json && \
	mv temp.json env.json
	@cd examples/sequential/publisher-with-ack && \
	jq '.redis.host = "localhost" | .history.mongo.host = "localhost"' ./env.json > temp.json && \
	mv temp.json env.json

	@echo "done!"

GOLANGCI_LINT_VERSION=v1.64.8
GOLANGCI_LINT_TOOL = $(GOPATH)/bin/golangci-lint

lint: ## run all the lint tools, install golangci-lint if not exist
	@if [ ! -x "$(GOLANGCI_LINT_TOOL)" ]; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) > /dev/null; \
	fi
	@$(if $(wildcard $(GOLANGCI_LINT_TOOL)),echo "Running golangci-lint...";) \
	$(GOLANGCI_LINT_TOOL)  run --fix -j 2 -v

FIELDALIGNMENT_TOOL = $(GOPATH)/bin/fieldalignment

vet: ## Field Alignment
	@if [ ! -x "$(FIELDALIGNMENT_TOOL)" ]; then \
		echo "Installing fieldalignment..."; \
		go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest; \
	fi
	@$(if $(wildcard $(FIELDALIGNMENT_TOOL)),echo "Running go vet with fieldalignment...";) \
	go vet -vettool=$(FIELDALIGNMENT_TOOL) ./... || exit 0

vet-fix: ## If fixed, the annotation for struct fields will be removed
	@if [ ! -x "$(FIELDALIGNMENT_TOOL)" ]; then \
		echo "Installing fieldalignment..."; \
		go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest; \
	fi
	@$(if $(wildcard $(FIELDALIGNMENT_TOOL)),echo "Running fieldalignment -fix...";) \
	$(FIELDALIGNMENT_TOOL) -fix ./... || exit 0

.PHONY: fuzz-generate
fuzz-generate: ## generate fuzz test data
	@echo "generate fuzz test file"
	@if ! command -v fzgen >/dev/null 2>&1; then \
		go install github.com/thepudds/fzgen/cmd/fzgen@latest; \
	fi
	go generate -x ./... || exit 0
	@echo "done!"

FUZZ_PATTERN = ^Fuzz_
FUZZ_TESTS = $(shell go list ./test/fuzz | xargs go test -list=$(FUZZ_PATTERN) | grep $(FUZZ_PATTERN))

.PHONY:$(FUZZ_TESTS)
$(FUZZ_TESTS):
	@echo "🚀 Running fuzz test: $@"
	go test -fuzz=$@ -fuzztime=10s

.PHONY:fuzz
fuzz:$(FUZZ_TESTS) ## start fuzzing test
	
.PHONY: help
help: ## show list of commands if they are with descriptions
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: delay delay-consumer delay-publisher normal normal-consumer normal-publisher \
 		sequential sequential-publisher sequential-consumer sequential-publisher-ack ui clean \
		lint vet vet-fix
