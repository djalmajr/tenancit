# syntax=docker/dockerfile:1

# --- Stage 1: build the SPA ---
FROM oven/bun:1.3-alpine AS web
WORKDIR /web
COPY web/package.json web/bun.lock* ./
RUN bun install --frozen-lockfile || bun install
COPY web/ ./
RUN bun run build

# --- Stage 2: build the Go binary with the SPA embedded ---
FROM golang:1.25-alpine AS server
WORKDIR /src
RUN apk add --no-cache git
ENV GOWORK=off
COPY server/ ./
RUN go mod download
# embed the built SPA
COPY --from=web /web/dist ./internal/spa/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# --- Stage 3: minimal runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/server /server
EXPOSE 8080
ENV RT_ADDR=:8080
ENTRYPOINT ["/server"]
