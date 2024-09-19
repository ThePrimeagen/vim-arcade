ARG GO_VERSION=1
FROM golang:1.23.0-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod /usr/src/app
RUN go mod download && go mod verify
COPY cmd cmd
COPY pkg pkg
RUN go build -v -o /run-app /usr/src/app/cmd/matchmaking/main.go

FROM debian:bookworm

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app"]
