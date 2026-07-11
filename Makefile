.PHONY: build build-web embed build-server lint lint-go lint-deploy test test-go test-go-strict test-web test-db \
        e2e-catalog e2e-smoke e2e-pr e2e e2e-oidc e2e-stability benchmark-scale sqlc \
        dev-server dev-web dev-compose dev-compose-up dev-compose-down tidy clean \
        docker docker-up docker-down docker-reset smoke deploy-preflight test-continuity test-postgres-roles

## build: web SPA -> embed into server -> Go binary
build: build-web embed build-server

build-web:
	cd web && bun install --frozen-lockfile && bun run build

## embed: copy the built SPA into the Go embed location
embed:
	rm -rf server/internal/spa/dist
	cp -r web/dist server/internal/spa/dist

build-server:
	cd server && go build -o bin/server ./cmd/server && go build -o bin/migrate ./cmd/migrate && go build -o bin/tenancit-rewrap ./cmd/tenancit-rewrap

## test: Go checks + web typecheck and unit tests. DB tests skip if Docker is unavailable.
test: test-go test-web

lint: lint-go lint-deploy
	cd web && bun run lint

lint-deploy:
	sh -n scripts/deploy-preflight.sh scripts/deploy-release.sh scripts/deploy-rollback.sh \
		deploy/postgres/configure-roles.sh scripts/test-deploy-scripts.sh \
		scripts/report-operation.sh scripts/postgres-backup.sh scripts/postgres-restore-drill.sh \
		scripts/test-multi-replica-continuity.sh scripts/test-postgres-roles.sh \
		scripts/post-deploy-production-smoke.sh scripts/test-production-smoke.sh \
		scripts/test-compose-contracts.sh
	sh ./scripts/test-deploy-scripts.sh
	sh ./scripts/test-production-smoke.sh
	sh ./scripts/test-compose-contracts.sh

lint-go:
	sh ./scripts/lint-go.sh

test-go: lint-go
	cd server && go test ./...

## test-go-strict: require DB integration tests instead of allowing skips.
# Set TENANCIT_TEST_DATABASE_URL to reuse a shared PostgreSQL service with an
# isolated database per test; otherwise the local fallback is Testcontainers.
test-go-strict: lint-go
	cd server && REQUIRE_DB_TESTS=1 go test ./...

test-web:
	cd web && bun run lint && bun run typecheck && bun run test

## test-db: compatibility alias for the canonical strict testcontainers gate.
test-db: test-go-strict

e2e-catalog:
	bun ./scripts/check-e2e-catalog.ts

## e2e-smoke: auth and all deep links in packaged + Vite/proxy modes.
e2e-smoke: e2e-catalog
	sh ./scripts/e2e.sh bootstrap.e2e.test.ts route-smoke.e2e.test.ts

## e2e-pr: PR-critical catalog tier, plus Vite route smoke.
e2e-pr: e2e-catalog
	sh ./scripts/e2e.sh --grep @pr-critical

## e2e: full catalog in packaged mode, plus Vite route smoke.
e2e: e2e-catalog
	sh ./scripts/e2e.sh

e2e-oidc:
	sh ./scripts/e2e-oidc.sh

## e2e-stability: full suite three times on fresh stacks, always without retry.
e2e-stability: e2e-catalog
	@for run in 1 2 3; do \
		echo "E2E stability run $$run/3"; \
		TENANCIT_E2E_RETRIES=0 sh ./scripts/e2e.sh || exit $$?; \
	done

## benchmark-scale: isolated 100/500/1000/5000 capacity curve and pagination gate.
benchmark-scale:
	sh ./scripts/benchmark-scale.sh

## sqlc: regenerate type-safe DB code (pinned version, runs outside the module)
sqlc:
	cd server && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate

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
	docker build -t tenancit:dev .

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

## docker-reset: explicitly destroy local containers AND persistent volumes.
docker-reset:
	@test "$(CONFIRM)" = "destroy-local-data" || (echo 'Refusing: use CONFIRM=destroy-local-data' >&2; exit 1)
	docker compose down -v

smoke:
	sh ./scripts/post-deploy-smoke.sh

deploy-preflight:
	sh ./scripts/deploy-preflight.sh

test-continuity:
	sh ./scripts/test-multi-replica-continuity.sh

test-postgres-roles: build-server
	sh ./scripts/test-postgres-roles.sh

clean:
	rm -rf server/bin web/dist server/internal/spa/dist
