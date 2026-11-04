.PHONY: build check dev web

web:
	cd web && pnpm build

check:
	cd web && pnpm check
	cd web && pnpm test
	go test ./...

build: web
	go build -o nyst .

dev:
	cd web && pnpm dev
