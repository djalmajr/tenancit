# --- Stage 1: build the SPA ---
FROM oven/bun:1.3-alpine AS web
WORKDIR /web
COPY web/package.json web/bun.lock* ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

# --- Stage 2: build the Go binary with the SPA embedded ---
FROM golang:1.26-alpine AS server
WORKDIR /src
RUN apk add --no-cache git
ENV GOWORK=off
COPY server/ ./
RUN go mod download
# embed the built SPA
COPY --from=web /web/dist ./internal/spa/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/tenancit-rewrap ./cmd/tenancit-rewrap
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/tenancit-audit-jobs ./cmd/tenancit-audit-jobs

# --- Stage 3: minimal runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/server /server
COPY --from=server /out/migrate /migrate
COPY --from=server /out/tenancit-rewrap /tenancit-rewrap
COPY --from=server /out/tenancit-audit-jobs /tenancit-audit-jobs
EXPOSE 8080
ENV TENANCIT_ADDR=:8080
CMD ["/server"]
