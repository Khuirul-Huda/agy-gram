.PHONY: all build run test clean pm2-start pm2-stop pm2-restart pm2-logs

BINARY_NAME=agy-gram
BIN_DIR=bin
MAIN_SRC=cmd/agy-gram/main.go

all: build

build:
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_SRC)
	@echo "Build successful: $(BIN_DIR)/$(BINARY_NAME)"

run: build
	./$(BIN_DIR)/$(BINARY_NAME)

test:
	go test -v ./...

clean:
	@rm -rf $(BIN_DIR)
	@echo "Cleaned build artifacts."

pm2-start: build
	pm2 start ecosystem.config.js

pm2-stop:
	pm2 stop agy-gram

pm2-restart: build
	pm2 restart agy-gram

pm2-logs:
	pm2 logs agy-gram
