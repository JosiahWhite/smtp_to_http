FROM golang:1.17-alpine3.14 AS builder

RUN apk add --no-cache git ca-certificates mailcap

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o smtp_to_http



FROM alpine:3.14

RUN apk add --no-cache ca-certificates mailcap

COPY --from=builder /app/smtp_to_http /smtp_to_http

USER daemon

ENV SMTP_ADDR="0.0.0.0:2525"
EXPOSE 2525


ENV HTTP_ADDR="0.0.0.0:8334"
EXPOSE 8334

ENTRYPOINT ["/smtp_to_http"]
