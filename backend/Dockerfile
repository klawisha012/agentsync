FROM golang:1.26.3-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/cmd cmd
COPY backend/internal internal
ARG CLI_VERSION=dev
RUN set -eu; \
    v=$(printf '%s' "$CLI_VERSION" | tr -cd 'A-Za-z0-9._+-'); \
    if [ -z "$v" ]; then v=dev; fi; \
    mkdir -p /cli-bin; \
    printf '%s\n' "$v" > /cli-bin/version; \
    ldflags="-X github.com/klawisha012/agentsync/internal/cli.Version=${v}"; \
    CGO_ENABLED=0 go build -ldflags "$ldflags" -o /server ./cmd/api; \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$ldflags" -o /cli-bin/agentsync-linux-amd64 ./cmd/agentsync; \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$ldflags" -o /cli-bin/agentsync-linux-arm64 ./cmd/agentsync; \
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$ldflags" -o /cli-bin/agentsync-windows-amd64.exe ./cmd/agentsync; \
    CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "$ldflags" -o /cli-bin/agentsync-windows-arm64.exe ./cmd/agentsync; \
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$ldflags" -o /cli-bin/agentsync-darwin-amd64 ./cmd/agentsync; \
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$ldflags" -o /cli-bin/agentsync-darwin-arm64 ./cmd/agentsync

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /server /server
COPY --from=build /cli-bin /cli-bin
COPY scripts /scripts
USER nobody
EXPOSE 8080
ENTRYPOINT ["/server"]
