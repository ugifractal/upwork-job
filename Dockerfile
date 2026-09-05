# syntax=docker/dockerfile:1
# check=skip=SecretsUsedInArgOrEnv

# Production build for Kamal. Minimal image, non-root user.

# Make sure GO_VERSION matches the version in go.mod
ARG GO_VERSION=1.25
FROM docker.io/library/golang:${GO_VERSION}-alpine AS build

WORKDIR /src

# Install application modules (cache-friendly layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy application code
COPY . .

# Build fully static binary; prepare /data owned by the runtime user for the bind mount
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/upwork-job . \
    && mkdir -p /data && chown 65532:65532 /data

# Final stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy built artifacts: binary + writable data dir (ownership preserved)
COPY --chown=65532:65532 --from=build /out/upwork-job /app/upwork-job
COPY --chown=65532:65532 --from=build /data /data

ENV HTTP_PORT=8080
EXPOSE 8080

CMD ["./upwork-job"]