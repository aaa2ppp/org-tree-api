FROM golang:1.25.11-alpine AS builder
WORKDIR /app
RUN apk --no-cache add make
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.27.1
RUN CGO_ENABLED=0 go install github.com/swaggo/swag/cmd/swag@v1.16.4
RUN CGO_ENABLED=0 go install golang.org/x/tools/cmd/stringer@latest

FROM alpine:3.23 AS migrate
WORKDIR /app
COPY --from=builder /go/bin/goose /bin
COPY scripts/migrate.sh .
COPY migrations migrations
ENTRYPOINT [ "sh", "migrate.sh", "up" ]

FROM builder AS server_builder
WORKDIR /app
COPY go.mod go.sum ./
RUN  go mod download
COPY pkg pkg
COPY cmd cmd
COPY internal internal
COPY Makefile .
RUN  CGO_ENABLED=0 make generate test bin/server

FROM scratch AS server
COPY --from=server_builder /app/bin/server .
EXPOSE 8080
USER 65535:65535
ENTRYPOINT [ "/server" ]
