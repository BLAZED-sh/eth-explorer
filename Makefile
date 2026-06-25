.PHONY: dev-server dev-web build run clean

dev-server:
	go run ./cmd/server

dev-web:
	cd web && npm run dev

build:
	cd web && npm run build
	CGO_ENABLED=0 go build -o blazed-explorer ./cmd/server

run: build
	./blazed-explorer

clean:
	rm -f blazed-explorer explorer.db explorer.db-wal explorer.db-shm
