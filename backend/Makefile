.PHONY: run build tidy migrate-up migrate-down

run:
	go run ./cmd/server/main.go

build:
	go build -ldflags "-s -w" -o bin/server ./cmd/server/main.go

tidy:
	go mod tidy

migrate-up:
	go run -mod=mod github.com/golang-migrate/migrate/v4/cmd/migrate \
		-path ./internal/db/migrations \
		-database "$(DATABASE_URL)" \
		up

migrate-down:
	go run -mod=mod github.com/golang-migrate/migrate/v4/cmd/migrate \
		-path ./internal/db/migrations \
		-database "$(DATABASE_URL)" \
		down 1

lint:
	golangci-lint run ./...

test:
	go test ./... -v
