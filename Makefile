all: go-generate

go-generate:
	cd pb && go generate
