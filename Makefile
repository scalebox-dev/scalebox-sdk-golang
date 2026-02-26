# Scalebox Go SDK Makefile

.PHONY: build test unit-test integration-test proto

build:
	go build ./...

test:
	go test ./...

# Run unit tests (excludes integration tests)
unit-test:
	go test ./...

# Run integration tests (requires SCALEBOX_BASE_URL and SCALEBOX_API_KEY)
integration-test:
	go test -tags integration ./integration_test/... -v

# Generate pb and pbconnect from proto. Requires protoc, protoc-gen-go, protoc-gen-connect-go.
# Install: brew install protobuf && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
proto:
	@command -v protoc >/dev/null 2>&1 || { echo "protoc not found; install with: brew install protobuf"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go not found; add \$$(go env GOPATH)/bin to PATH and run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@command -v protoc-gen-connect-go >/dev/null 2>&1 || { echo "protoc-gen-connect-go not found; add \$$(go env GOPATH)/bin to PATH and run: go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest"; exit 1; }
	protoc -I /opt/homebrew/include -I . \
		--go_out=. --go_opt=module=github.com/scalebox/scalebox-sdk-golang \
		--connect-go_out=. --connect-go_opt=module=github.com/scalebox/scalebox-sdk-golang \
		proto/api.proto
