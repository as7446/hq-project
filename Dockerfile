# syntax=docker/dockerfile:1.7

FROM golang:1.23-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod ./
RUN  go mod download

COPY . .
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /out/hq-project ./cmd/server


FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /out/hq-project /app/hq-project

ENV PORT=8080 \
    SERVICE_NAME=hq-project-demo \
    APP_ENV=prod \
    TZ=Asia/Shanghai

EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/app/hq-project", "--healthcheck"]
ENTRYPOINT ["/app/hq-project"]

