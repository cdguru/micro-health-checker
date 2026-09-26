# syntax=docker/dockerfile:1.7
FROM golang:1.27.1-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -trimpath \
    -ldflags="-s -w -X github.com/cdguru/micro-health-checker/internal/version.Version=${VERSION} -X github.com/cdguru/micro-health-checker/internal/version.Commit=${COMMIT} -X github.com/cdguru/micro-health-checker/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/micro-health-checker ./cmd/micro-health-checker && \
    mkdir -p /out/data /out/config

FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.opencontainers.image.title="micro-health-checker" \
      org.opencontainers.image.description="Protocol-to-REST health gateway for databases, brokers and infrastructure services" \
      org.opencontainers.image.source="https://github.com/cdguru/micro-health-checker" \
      org.opencontainers.image.licenses="Apache-2.0"
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chown=65532:65532 /out/data /data
COPY --from=build --chown=65532:65532 /out/config /etc/micro-health-checker
COPY --from=build --chown=65532:65532 /out/micro-health-checker /micro-health-checker
USER 65532:65532
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/micro-health-checker"]
CMD ["-config", "/etc/micro-health-checker/config.yml"]
