.PHONY: build build-web embed build-server test test-go test-web test-db sqlc \
        dev-server dev-web dev-compose dev-compose-up dev-compose-down tidy clean \
        pg-test-up pg-test-down docker docker-up docker-down

## build: web SPA -> embed into server -> Go binary
build: build-web embed build-server

build-web:
	cd web && bun install && bun run build

## embed: copy the built SPA into the Go embed location
embed:
	rm -rf server/internal/spa/dist
	cp -r web/dist server/internal/spa/dist

build-server:
	cd server && go build -o bin/server ./cmd/server

## test: unit tests (Go + web typecheck). Use test-db for DB-backed tests.
test: test-go test-web

test-go:
	cd server && go vet ./... && go test ./...

test-web:
	cd web && bunx tsc --noEmit

## test-db: Go tests against an ephemeral Postgres
test-db: pg-test-up
	cd server && TEST_DATABASE_URL="postgres://postgres:test@localhost:55432/konvario_test?sslmode=disable" go test ./...
	$(MAKE) pg-test-down

pg-test-up:
	docker rm -f konvario-pg-test >/dev/null 2>&1 || true
	docker run -d --name konvario-pg-test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=konvario_test -p 55432:5432 postgres:16-alpine >/dev/null
	@for i in $$(seq 1 30); do docker exec konvario-pg-test pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done

pg-test-down:
	docker rm -f konvario-pg-test >/dev/null 2>&1 || true

## sqlc: regenerate type-safe DB code (pinned version, runs outside the module)
sqlc:
	cd server && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate

dev-server:
	cd server && go run ./cmd/server

dev-web:
	cd web && bun run dev

dev-compose:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up postgres api-dev web-dev

dev-compose-up:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d postgres api-dev web-dev

dev-compose-down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

tidy:
	cd server && go mod tidy

docker:
	docker build -t konvario:dev .

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

clean:
	rm -rf server/bin web/dist server/internal/spa/dist
