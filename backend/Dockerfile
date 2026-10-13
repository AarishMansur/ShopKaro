# syntax=docker/dockerfile:1
# Build context is the repo root (works for docker compose, Render blueprint
# and Render's default dashboard builds).
FROM golang:1.27-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 shopkaro
COPY --from=build /out/api /app/api
COPY --from=build /out/seed /app/seed
USER shopkaro
WORKDIR /app
EXPOSE 8080
# Seed is idempotent: first boot loads the demo kirana, later boots skip.
CMD ["/bin/sh", "-c", "/app/seed && exec /app/api"]
