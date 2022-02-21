IMG ?= rafay-prompt:latest

.PHONY: test
test:  ## Run the tests.
	@$(GO) test ./...


.PHONY: code-gen
code-gen: ## Generate source codes.
	./_tools/codegen.sh


.PHONY: tidy
tidy:
	GOPROXY=direct GOPRIVATE=github.com/RafaySystems/* go mod tidy

build:
	docker build . -t ${IMG} --build-arg BUILD_USR=${BUILD_USER} --build-arg BUILD_PWD=${BUILD_PASSWORD}	

generate:
	go generate ./...

.PHONY: check
check:
	go fmt ./...
	go vet ./...

gen-proto:
	$(MAKE) vendor

	protoc \
	-I=vendor/github.com/gogo/protobuf/gogoproto \
	-I=proto/rpc/v2 \
	-I=vendor/ \
	-I=../../../ \
	-I=vendor/github.com/grpc-ecosystem/grpc-gateway/ \
	-I=vendor/github.com/gogo/googleapis/ \
	-I=vendor/github.com/RafaySystems/cluster-controller/api/v2 \
	--gogofaster_out=plugins=grpc,\
	Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
	Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:proto/rpc/v2 \
	--grpc-gateway_out=allow_patch_feature=false,\
	Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
	Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:proto/rpc/v2 \
	--swagger_out=logtostderr=true:api/def/v2 \
	proto/rpc/v2/*.proto

	go generate ./...
