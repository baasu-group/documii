# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS build
WORKDIR /src/services/core
COPY services/core/go.mod services/core/go.sum ./
RUN go mod download
COPY services/core/ ./
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -o /out/worker ./cmd/worker \
    && CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 go build -trimpath -o /out/admin ./cmd/admin

FROM alpine:3.21
RUN addgroup -S documi && adduser -S -G documi documi
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
COPY database/migrations/ /app/migrations/
USER documi
EXPOSE 8080
ENTRYPOINT ["api"]
