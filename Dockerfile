############################################
# Builder: Go 1.24, static, netgo
############################################
FROM --platform=$BUILDPLATFORM docker.io/golang:1.24-alpine AS build

WORKDIR /src

# Trust store + tzdata for copying into final image
RUN apk --no-cache add ca-certificates tzdata

ARG BINARY_NAME=app
ARG BUILD_PACKAGE

# Deterministic, static build (pure Go DNS via netgo)
ENV CGO_ENABLED=0 \
    GOFLAGS="-trimpath" \
    GOTOOLCHAIN=local

# Pre-fetch modules (better layer caching)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Build
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -tags netgo \
      -ldflags='-s -w -extldflags "-static"' \
      -o /out/$BINARY_NAME \
      ${BUILD_PACKAGE}

############################################
# Final: tiny scratch image
############################################
FROM scratch

# Run as non-root (no /etc/passwd needed when using numeric UID)
USER 48000:48000

# Bring in the binary
ARG BINARY_NAME=app
COPY --from=build /out/${BINARY_NAME} /${BINARY_NAME}
EXPOSE 8080

# Default entrypoint
ENTRYPOINT ["/app"]
