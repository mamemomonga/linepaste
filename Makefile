BIN := linepaste
# PREFIX := /usr/local/bin
PREFIX := $(HOME)/bin

.PHONY: build install uninstall clean tidy

build: tidy
	go build -o $(BIN) .

tidy:
	go mod tidy

install: build
	install -m 755 $(BIN) $(PREFIX)/$(BIN)

uninstall:
	rm -f $(PREFIX)/$(BIN)

clean:
	rm -f $(BIN)
