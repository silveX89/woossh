.PHONY: build test smoke vet clean all

BINARY=woossh

build:
	go build -o $(BINARY) .

test:
	go test ./... -v

smoke: build
	scripts/smoke.sh

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

all: vet test smoke
	@echo "══ All checks passed ══"