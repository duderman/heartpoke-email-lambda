.PHONY: clean

HEARTPOKE_PATH = ../heartpoke/cf/lib/book

build:
	GOOS=linux go build -o bin/main main.go

copy:
	mkdir -p ${HEARTPOKE_PATH}
	cp bin/main ${HEARTPOKE_PATH}

clean:
	rm bin/main

.DEFAULT_GOAL :=
all: build copy clean