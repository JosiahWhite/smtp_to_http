# SMTP to HTTP

[![Docker Hub](https://img.shields.io/docker/pulls/josiahwhite/smtp_to_http.svg?style=flat-square)][Docker Hub]
[![Go Report Card](https://goreportcard.com/badge/github.com/JosiahWhite/smtp_to_http?style=flat-square)][Go Report Card]
[![License](https://img.shields.io/github/license/JosiahWhite/smtp_to_http.svg?style=flat-square)][License]

[Docker Hub]:      https://hub.docker.com/r/josiahwhite/smtp_to_http
[Go Report Card]:  https://goreportcard.com/report/github.com/JosiahWhite/smtp_to_http
[License]:         https://github.com/JosiahWhite/smtp_to_http/blob/main/LICENSE

`smtp_to_http` is an application that listens for SMTP and stores all email for a configurable period.

This can be used as an API for providing temporary email addresses similar to guerrillamail or temp-mail but powered by your own domain and giving easy API access.

## Getting started

```
docker run --name smtp_to_http -e SMTP_PRIMARY_HOST=example.com -e MAIL_EXPIRE_DURATION=10m josiahwhite/smtp_to_http
```

By default the container listens on port 2525 for SMTP and 8334 for HTTP access.

## API Access

The HTTP API provides access through the following requests:

To clear messages for an inbox:
```
curl 'http://127.0.0.1:8334/clearMessages?email=cool%40example.com'
```

To read messages for an inbox:
```
curl 'http://127.0.0.1:8334/fetchMessages?email=cool%40example.com'
```