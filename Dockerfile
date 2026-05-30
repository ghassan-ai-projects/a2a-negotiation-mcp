# Stage 1: Build
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server/

# Stage 2: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/server /app/server
COPY data/seeds/saas_pricing.csv /app/data/seeds/saas_pricing.csv
EXPOSE 8080
ENTRYPOINT ["/app/server"]
CMD ["-seed", "/app/data/seeds/saas_pricing.csv"]
