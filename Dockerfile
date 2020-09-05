FROM golang:1.15 AS builder

RUN mkdir /app
WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .
RUN make build

FROM alpine:3.12

LABEL MAINTAINER="kazhuravlev@fastmail.com"

ENV APP_DIR /app
ENV BIN_FILE ${APP_DIR}/leader-election
ENV CONFIG_FILE ${APP_DIR}/config/config.toml

RUN mkdir -p ${APP_DIR}

COPY --from=builder /app/bin/leader-election ${BIN_FILE}

CMD ${BIN_FILE} -c ${CONFIG_FILE} run
