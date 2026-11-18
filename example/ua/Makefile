.PHONY: all build bench check

all: build

build:
	go build -o build src/build/main.go
	go build -o check src/check/main.go
	go build -o bench src/bench/main.go

bench:
	go run src/bench/main.go \
	-train-pos ../datasets/ua/train_pos.txt \
	-train-neg ../datasets/ua/train_neg.txt \
	-test-pos  ../datasets/ua/test_pos.txt \
	-test-neg  ../datasets/ua/test_neg.txt -loops 100 -epochs 5 -lr 0.5
