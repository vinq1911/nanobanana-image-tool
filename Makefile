BUILD_DIR := ./bin

.PHONY: build build-mcp run test clean serve generate-example

build:
	go build -o $(BUILD_DIR)/nanobanana-tool ./cmd/nanobanana-tool
	go build -o $(BUILD_DIR)/nanobanana-mcp ./cmd/nanobanana-mcp

build-mcp:
	go build -o $(BUILD_DIR)/nanobanana-mcp ./cmd/nanobanana-mcp

run: build
	$(BUILD_DIR)/nanobanana-tool serve

test:
	go test ./... -v

clean:
	rm -rf $(BUILD_DIR) ./output

serve: build
	$(BUILD_DIR)/nanobanana-tool serve

generate-example: build
	$(BUILD_DIR)/nanobanana-tool generate \
		--prompt "friendly robot in a colorful town, children's book illustration" \
		--width 1024 \
		--height 1024 \
		--output ./output
