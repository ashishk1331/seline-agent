# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache module downloads across builds.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
# Static, stripped binary: CGO off (no libc), -s -w drops the symbol table and
# DWARF debug info, -trimpath removes local paths. ~25-30% smaller binary.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath -ldflags="-s -w" \
      -o /out/agent ./cmd/agent

# ---- runtime stage ----
# Alpine (not scratch/distroless) because the run_command tool needs /bin/sh.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
 && adduser -D -u 10001 seline \
 && mkdir -p /app/workspace \
 && chown -R seline:seline /app

COPY --from=build /out/agent /usr/local/bin/agent

USER seline
WORKDIR /app
ENV WORKSPACE_DIR=/app/workspace
ENTRYPOINT ["agent"]
