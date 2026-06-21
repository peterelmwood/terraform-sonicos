BINARY  := terraform-provider-sonicos
VERSION ?= dev

.PHONY: build install test vet fmt tidy clean

# Build the provider binary.
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

# Run the unit test suite.
test:
	go test ./... -count=1

# Static analysis.
vet:
	go vet ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

# Build and place the binary in the Terraform CLI plugin override path so
# `terraform plan`/`apply` can find a locally built provider. Point a
# dev_overrides block in your CLI config at $(GOBIN) to use it.
install: build
	mkdir -p $(shell go env GOBIN 2>/dev/null || echo $(HOME)/go/bin)
	cp $(BINARY) $(shell go env GOBIN 2>/dev/null || echo $(HOME)/go/bin)/

clean:
	rm -f $(BINARY)
