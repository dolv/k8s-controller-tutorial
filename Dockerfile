# syntax=docker/dockerfile:1.4
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -v -o k8s-controller-tutorial -ldflags "-X=github.com/dolv/k8s-controller-tutorial/cmd.appVersion=$VERSION" main.go


# Final stage
FROM nginx:1.28.0 AS nginx
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /app/jaegernginxproxy-controller .
COPY --from=builder /app/swagger /swagger
COPY --from=nginx /usr/sbin/nginx /usr/sbin/nginx
EXPOSE 8080
ENTRYPOINT ["/k8s-controller-tutorial"] 