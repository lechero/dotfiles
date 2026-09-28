BIN := manage-disk
PREFIX ?= $(HOME)/.local/bin

.PHONY: build run dry report test install clean

build:
	go build -o $(BIN) .

run: build
	./$(BIN)

dry: build
	./$(BIN) --dry-run

report: build
	./$(BIN) report

test:
	go vet ./...
	go test -race ./...

install: build
	mkdir -p $(PREFIX)
	cp $(BIN) $(PREFIX)/$(BIN)
	@echo "installed $(PREFIX)/$(BIN)"

clean:
	rm -f $(BIN)
