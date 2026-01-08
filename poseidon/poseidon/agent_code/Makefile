# Poseidon Agent Build System
# Uses the unified config builder tool

BUILDER=go run ./cmd/builder
CONFIG_DIR=./cmd/builder/testdata

BINARY_NAME=poseidon

# Protocol buffer generation (DNS profile)
build_protobuf_go:
	protoc --go_out=`pwd`/pkg/profiles/dnsgrpc --go_opt=paths=source_relative \
--go-grpc_out=`pwd`/pkg/profiles/dnsgrpc --go-grpc_opt=paths=source_relative \
--proto_path=`pwd`/pkg/profiles/dnsgrpc \
`pwd`/pkg/profiles/dnsgrpc/*.proto

# Build targets using the unified config builder
build_http:
	${BUILDER} --config ${CONFIG_DIR}/http-test.json

build_websocket:
	${BUILDER} --config ${CONFIG_DIR}/websocket-test.json

build_tcp:
	${BUILDER} --config ${CONFIG_DIR}/tcp-test.json

# Validate config without building
validate_http:
	${BUILDER} --config ${CONFIG_DIR}/http-test.json --validate

validate_websocket:
	${BUILDER} --config ${CONFIG_DIR}/websocket-test.json --validate

validate_tcp:
	${BUILDER} --config ${CONFIG_DIR}/tcp-test.json --validate

# Dry run to see what would happen
dryrun_http:
	${BUILDER} --config ${CONFIG_DIR}/http-test.json --dry-run

# Run targets
run_http:
	./${BINARY_NAME}_http.bin

run_websocket:
	./${BINARY_NAME}_websocket.bin

run_tcp:
	./${BINARY_NAME}_tcp

# Build and run
build_and_run_http: build_http run_http

build_and_run_websocket: build_websocket run_websocket

build_and_run_tcp: build_tcp run_tcp

# Build all profiles
build_all: build_http build_websocket build_tcp

# Clean
clean:
	go clean
	rm -f ${BINARY_NAME}_*.bin ${BINARY_NAME}_tcp
	rm -f pkg/config/config.go

# Build the builder tool itself
build_builder:
	go build -o builder.exe ./cmd/builder

# Help
help:
	@echo "Poseidon Agent Build Targets:"
	@echo ""
	@echo "  build_http       - Build agent with HTTP profile"
	@echo "  build_websocket  - Build agent with Websocket profile"
	@echo "  build_tcp        - Build agent with TCP profile"
	@echo "  build_all        - Build all profiles"
	@echo ""
	@echo "  validate_*       - Validate config without building"
	@echo "  dryrun_*         - Show what would happen without building"
	@echo ""
	@echo "  run_*            - Run the built binary"
	@echo "  build_and_run_*  - Build and run"
	@echo ""
	@echo "  clean            - Remove built binaries and generated config"
	@echo "  build_builder    - Build the builder tool itself"
	@echo ""
	@echo "  build_protobuf_go - Generate DNS protobuf code"
	@echo ""
	@echo "Config files are in: ${CONFIG_DIR}"
	@echo "Edit these JSON files to customize agent configuration."

.PHONY: build_http build_websocket build_tcp build_all \
        validate_http validate_websocket validate_tcp \
        run_http run_websocket run_tcp \
        build_and_run_http build_and_run_websocket build_and_run_tcp \
        clean build_builder build_protobuf_go help
