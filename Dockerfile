FROM golang:1.13.4-alpine AS build
RUN apk add build-base

WORKDIR /src
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -mod=vendor -o beacon main.go

FROM alpine:3.10
RUN apk update && apk add --no-cache ca-certificates

COPY --from=build /src/beacon /usr/bin
COPY ./scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["agent"]