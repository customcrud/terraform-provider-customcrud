default: fmt lint install generate

build:
	go build -o terraform-provider-customcrud
	cp terraform-provider-customcrud ~/.terraform.d/plugins/registry.terraform.io/customcrud/customcrud/1.0.0/linux_amd64/
	rm -f examples/.terraform.lock.hcl
	cd examples && terraform init

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
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: fmt lint test testacc build install generate
