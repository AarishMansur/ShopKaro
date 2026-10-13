# syntax=docker/dockerfile:1
FROM golang:1.27-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
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
