default: fmt lint install generate

local_registry_path=$(HOME)/.terraform.d/plugins/registry.terraform.io/customcrud/customcrud/1.0.0/$(shell go env GOOS)_$(shell go env GOARCH)
build:
	go build -o terraform-provider-customcrud
	mkdir -p "$(local_registry_path)"
	cp terraform-provider-customcrud "$(local_registry_path)/terraform-provider-customcrud"
	cd examples/file && rm -f .terraform.lock.hcl && terraform init

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m -coverprofile=coverage.out -covermode=atomic ./...

.PHONY: fmt lint test testacc build install generate
