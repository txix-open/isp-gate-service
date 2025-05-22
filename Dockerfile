FROM golang:1.24-alpine3.20 AS builder
WORKDIR /build
ARG version
ENV version_env=$version
ARG app_name
ENV app_name_env=$app_name
COPY . .
RUN go build -ldflags="-X 'main.version=$version_env'" -o /main .

FROM alpine:3.20 AS runner

RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Europe/Moscow /etc/localtime && \
    echo "Europe/Moscow" > /etc/timezone

ARG UID=10001
ARG GID=10001

RUN addgroup -g ${GID} appuser
RUN adduser \
    -D \
    -h "/nonexistent" \
    -s "/sbin/nologin" \
    -u "${UID}" \
    -G appuser \
    appuser

RUN mkdir -p /app/data
RUN chown  appuser:appuser /app/data
VOLUME /app/data

USER appuser

WORKDIR /app

ARG app_name
ENV app_name_env=$app_name
COPY --from=builder main /app/$app_name_env
COPY /conf/config.yml /app/config.yml
COPY /conf/default_remote_config.json* /app/default_remote_config.json

ENTRYPOINT /app/$app_name_env
