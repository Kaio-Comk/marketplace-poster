.PHONY: build test vet

build:
	go build -o marketplace-poster.exe .

test:
	go vet ./... && go test ./...

vet:
	go vet ./...
