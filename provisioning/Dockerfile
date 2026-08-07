FROM golang:1.25.0-alpine AS builder

ARG BIN=go-mud-server
ENV BIN=${BIN}
WORKDIR /src

RUN apk add --update make bash

# Create logfile
RUN mkdir -p /app/log
RUN touch /app/log/logfile.log

COPY . .
RUN go generate
RUN go build -o ${BIN}

FROM golang:1.25.0-bookworm AS test

ENV CGO_ENABLED=1
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go generate ./...

CMD ["go", "test", "-v", "-count=1", "-timeout", "300s", "-race", "./..."]

FROM alpine:latest AS runner
ARG BIN=go-mud-server
ENV BIN=${BIN}
WORKDIR /app

EXPOSE ${PORT}
COPY --from=builder /src/${BIN} .
COPY --from=builder /src/_datafiles ./_datafiles

ENTRYPOINT ["/bin/sh", "-c", "./${BIN}"]