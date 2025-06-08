# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o k8s-controller-tutorial main.go

# Final stage
FROM gcr.io/distroless/base-debian11
WORKDIR /
COPY --from=builder /app/k8s-controller-tutorial .
EXPOSE 8080
ENTRYPOINT ["/k8s-controller-tutorial"] 