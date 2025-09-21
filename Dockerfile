# --- build stage ---
FROM golang:alpine AS builder
WORKDIR /app

# deps
COPY go.mod go.sum ./
RUN go mod download

# source
COPY . .

# build binarke
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o follower-service main.go

# --- runtime stage ---
FROM alpine:latest
WORKDIR /app
# (opciono, korisno za TLS i sl.)
RUN apk --no-cache add ca-certificates

# kopiraj binarku
COPY --from=builder /app/follower-service .

# gRPC port
EXPOSE 50051

# default adresa (možeš pregaziti u compose-u)
ENV FOLLOWER_SERVICE_ADDRESS=:50051

# start
CMD ["./follower-service"]
