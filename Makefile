BIN := emoji-transfer

.PHONY: build install clean

build:
	go build -o $(BIN) ./cmd/main.go

install:
	go build -o /usr/local/bin/$(BIN) ./cmd/main.go

clean:
	rm -f $(BIN)
