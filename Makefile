BINARY := lazy-fluxcd

.PHONY: build run tidy lint clean

build:
	go build -o $(BINARY) .

run:
	go run .

tidy:
	go mod tidy

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
