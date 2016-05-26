default: test

updatedeps:
	go get -v -u github.com/tools/godep
	go get -v -u ./...
	godep save

deps:
	go get -v github.com/tools/godep

build: deps 
	godep go build -o bin/kafka-firehose-nozzle

test: deps
	godep go test -v -parallel 5

bench:
	godep go test -bench .

test-race:
	godep go test -v -race -parallel 5

test-all: vet lint test cover

vet:
	@go get golang.org/x/tools/cmd/vet
	go tool vet *.go

lint:
	@go get github.com/golang/lint/golint
	golint ./...

# cover shows test coverages
cover:
	@go get golang.org/x/tools/cmd/cover		
	godep go test -coverprofile=cover.out
	go tool cover -html cover.out
	rm cover.out

generate:
	go generate
