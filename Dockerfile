FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /onelab-api .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
RUN adduser -D appuser
USER appuser
WORKDIR /
COPY --from=builder /onelab-api /onelab-api
EXPOSE 8080
CMD ["/onelab-api"]