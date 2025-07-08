FROM ubuntu:latest
LABEL authors="akozadaev"

ENTRYPOINT ["top", "-b"]

FROM golang:1.23.4-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o oauth2-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/oauth2-server .
COPY --from=builder /app/migrations ./migrations
CMD ["./oauth2-server"]