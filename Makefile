GOPATH=$(shell go env GOPATH)
SUITE_TEST_FILES := $(shell find . -type f -name "*_suite_test.go")

.PHONY: test
test:
	@echo "start test"
	@set -e;\
	docker compose down -v || true
	docker compose up -d --build
	sleep 10
	go test -race -coverprofile=coverage.txt ./...

.PHONY: clean-docker-compose
clean-docker-compose:
	@echo "clean docker compose"
	@docker compose down -v || true

.PHONY: clean-test
clean-test:
	@echo "clean test"
	@docker compose down

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

ui:
	@echo "start ui on port:9090"
	@cd examples/ui && \
	jq '.redis.host = "redis-beanq" | .history.mongo.host = "mongo-beanq"' ./env.json > temp.json && \
	mv temp.json env.json && \
	go run -race ./main.go

clean:
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

GOLANGCI_LINT_VERSION=v2.2.1
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
 		sequential sequential-publisher sequential-consumer sequential-publisher-ack ui clean \
		lint vet vet-fix
