lint: protolint golint

PROTO_FILES := proto/words/words.proto proto/update/update.proto proto/search/search.proto

protobuf:
	PATH="$(PATH):$(HOME)/go/bin" \
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)
protolint:
	protolint .

golint:
	golangci-lint run -E gocritic -v ./...