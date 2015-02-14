all: go-generate

go-generate:
	@if type protoc > /dev/null; then cd pb && go generate; else echo "Protoc not available. Skipping go generate. Install protobuf-compiler."; fi
	go get -v .
	go install -v .
	go test -v ./...