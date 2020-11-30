FROM golang:1.14.3-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.* .
RUN go mod download
COPY . .
RUN go build -o /out/serv .


FROM alpine:3.12 AS alp
RUN apk add --no-cache ffmpeg ffmpeg-libs vidstab --repository http://dl-cdn.alpinelinux.org/alpine/edge/community
COPY --from=build /out/serv /
RUN mkdir WORKDIR
