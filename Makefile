.PHONY: test build server client panic security-check iso

test:
	go test ./...

build:
	./tools/build-binaries.sh

server:
	go run ./cmd/deaddrop-server -db ./drops.dev.db

client:
	go run ./cmd/deaddrop-client

panic:
	go run ./cmd/deaddrop-panic

security-check:
	python3 tests/security/check_static_invariants.py

iso: build
	./iso/build-iso.sh
