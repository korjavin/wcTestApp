BINARY_NAME=wctestapp
MAIN_PACKAGE=./cmd/wctestapp

.PHONY: all build clean test run lint

all: test build

build:
	go build -o $(BINARY_NAME) $(MAIN_PACKAGE)

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

test-coverage:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html

run:
	go run $(MAIN_PACKAGE)

lint:
	golangci-lint run

deps:
	go mod download

tidy:
	go mod tidy