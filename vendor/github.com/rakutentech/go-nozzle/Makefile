default: test

updatedeps:
	go get -v -u ./...

deps:
	go get -v ./...

test: deps
	go test -v -parallel 5

test-race: deps
	go test -v -race -parallel 5

test-all: vet lint test test-race

vet: deps
	go vet *.go

lint: deps
	@go get github.com/golang/lint/golint
	golint ./...

# cover shows test coverages
cover: deps
	@go get golang.org/x/tools/cmd/cover		
	go test -coverprofile=cover.out
	go tool cover -html cover.out
	rm cover.out
