FROM --platform=$BUILDPLATFORM golang:1.26.3-alpine AS builder
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./

RUN go mod download

# OPERATOR_NAME is declared after go mod download to preserve layer caching.
# Moving it earlier would invalidate the download cache for each operator build.
ARG OPERATOR_NAME

COPY api/ api/
COPY cmd/${OPERATOR_NAME}/ cmd/${OPERATOR_NAME}/
COPY internal/ internal/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -a -o ${OPERATOR_NAME}-operator cmd/${OPERATOR_NAME}/main.go

FROM gcr.io/distroless/static:nonroot
ARG OPERATOR_NAME
WORKDIR /
COPY --from=builder /workspace/${OPERATOR_NAME}-operator /operator
USER 65532:65532

ENTRYPOINT ["/operator"]
