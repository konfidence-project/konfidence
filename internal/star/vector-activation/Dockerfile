# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Install git and ca-certificates for private repo access
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./

# Download dependencies with GitHub token from BuildKit secret
ENV GOPRIVATE=github.com/konfidence-project/*
RUN --mount=type=secret,id=gh_token \
    GH_TOKEN=$(cat /run/secrets/gh_token) && \
    git config --global url."https://x-access-token:${GH_TOKEN}@github.com/".insteadOf "https://github.com/" && \
    go mod download && \
    git config --global --unset url."https://x-access-token:${GH_TOKEN}@github.com/".insteadOf


COPY cmd/ cmd/
COPY internal/ internal/


RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -a -o manager cmd/main.go

# Run stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
