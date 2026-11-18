.PHONY: all build bench check

all: build

build:
	go build -o build src/build/main.go
	go build -o check src/check/main.go
	go build -o bench src/bench/main.go

bench:
	go run src/bench/main.go \
	-train-pos ../datasets/ip/train_pos.txt \
	-train-neg ../datasets/ip/train_neg.txt \
	-test-pos  ../datasets/ip/test_pos.txt \
	-test-neg  ../datasets/ip/test_neg.txt -loops 100 -epochs 5 -lr 0.5
