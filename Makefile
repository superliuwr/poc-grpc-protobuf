SERVER_OUT := "bin/server"
CLIENT_OUT := "bin/client"
API_OUT := "customer/customer.pb.go"
PKG := "poc-grpc-protobuf-go"
SERVER_PKG_BUILD := "${PKG}/server"
CLIENT_PKG_BUILD := "${PKG}/client"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)

.PHONY: all api build_server build_client

all: build_server build_client

api/customer.pb.go: customer/customer.proto
	@protoc -I customer/ \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:customer \
		customer/customer.proto

api/customer.pb.gw.go: customer/customer.proto
	@protoc -I customer/ \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:customer \
		customer/customer.proto

api/customer.swagger.json: customer/customer.proto
	@protoc -I customer/ \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--swagger_out=logtostderr=true:customer \
		customer/customer.proto

api: api/customer.pb.go api/customer.pb.gw.go api/customer.swagger.json

dep: ## Get the dependencies
	@go get -v -d ./...

build_server: dep api ## Build the binary file for server
	@go build -i -v -o $(SERVER_OUT) $(SERVER_PKG_BUILD)

build_client: dep api ## Build the binary file for client
	@go build -i -v -o $(CLIENT_OUT) $(CLIENT_PKG_BUILD)

clean: ## Remove previous builds
	@rm $(SERVER_OUT) $(CLIENT_OUT) $(API_OUT)

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
