FROM golang:1.25-alpine AS build

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/supportflow ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S supportflow \
    && adduser -S -G supportflow supportflow \
    && mkdir -p /var/lib/supportflow/storage \
    && chown -R supportflow:supportflow /var/lib/supportflow

COPY --from=build /out/supportflow /usr/local/bin/supportflow
COPY --from=build /out/migrate /usr/local/bin/supportflow-migrate

USER supportflow
WORKDIR /var/lib/supportflow
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/supportflow"]
