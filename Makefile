.PHONY: build run clean install

build:
	go build -o clashtui .

run:
	go run .

clean:
	rm -f clashtui

install:
	go install .