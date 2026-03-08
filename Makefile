BINARY_NAME := nanobanana-tool
BUILD_DIR := ./bin
CMD_DIR := ./cmd/nanobanana-tool

.PHONY: build run test clean serve generate-example

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

run: build
	$(BUILD_DIR)/$(BINARY_NAME) serve

test:
	go test ./... -v

clean:
	rm -rf $(BUILD_DIR) ./output

serve: build
	$(BUILD_DIR)/$(BINARY_NAME) serve

generate-example: build
	$(BUILD_DIR)/$(BINARY_NAME) generate \
		--prompt "friendly robot in a colorful town, children's book illustration" \
		--width 1024 \
		--height 1024 \
		--output ./output
