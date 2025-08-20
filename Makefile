TAG ?= latest

.PHONY: init
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/golang/mock/mockgen@latest

.PHONY:mock
mock:
	mockgen -source=./pkg/codegraph/store/storage.go -destination=./test/mocks/mock_graph_store.go -package=mocks
	mockgen -source=./internal/repository/workspace.go -destination=./test/mocks/mock_workspace_repository.go -package=mocks
	mockgen -source=./internal/repository/event.go -destination=test/mocks/mock_event_repository.go -package=mocks
	mockgen -source=./internal/service/indexer.go -destination=test/mocks/mock_indexer.go -package=mocks
	mockgen -source=./pkg/codegraph/workspace/workspace.go -destination=test/mocks/mock_workspace.go -package=mocks
.PHONY:proto
proto:
	protoc --go_out=. pkg/codegraph/proto/file_element.proto
	protoc --go_out=. pkg/codegraph/proto/symbol_definition.proto
	protoc --go_out=. pkg/codegraph/proto/types.proto
	protoc --go_out=. pkg/codegraph/proto/test_message.proto

.PHONY:test
test:
	go test ./internal/...
	go test ./test/codegraph/...

api-test:
	@echo "Running API tests, make sure the server is started on port 11380"
	go test ./test/api/...

.PHONY:build
build:
	go mod tidy
	go build -ldflags="-s -w" -o ./bin/main ./cmd/main.go

.PHONY: swag
swag:
	swag init -g cmd/main.go -o docs/swagger

.PHONY: swag-ui
swag-ui:
	mkdir -p docs/swagger-ui
	cp -r $$(go list -f '{{.Dir}}' -m github.com/swaggo/swag)/example/docs/swagger-ui/* docs/swagger-ui/

.PHONY: docs
docs: swag swag-ui
	@echo "Swagger documentation generated successfully"
	@echo "Access the documentation at: http://localhost:8080/docs"
