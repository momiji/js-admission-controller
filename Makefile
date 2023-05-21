BIN = js-admission-controller
SRC = $(shell find -type f -name '*.go')

.PHONY: all
all: $(BIN)

$(BIN): $(SRC)
	go build -o $(BIN)

.PHONY: run
run: $(BIN)
	./$(BIN)

.PHONY: clean
clean:
	rm $(BIN)
