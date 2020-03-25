FROM golang:alpine AS build

RUN apk add libusb-dev gcc musl-dev --no-cache
RUN mkdir /build
WORKDIR /build
# RUN which gcc
# RUN echo $PATH
RUN ls /usr/include
COPY . .
RUN go build ./cmd/multiserver/*.go
