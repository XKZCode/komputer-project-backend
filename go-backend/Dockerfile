# Stage 1: Build binary Go
FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Stage 2: Runtime image super ringan
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
RUN mkdir -p uploads

EXPOSE 8080
CMD ["./main"]