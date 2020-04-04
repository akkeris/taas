FROM golang:1.12-alpine
RUN apk update
RUN apk add openssl ca-certificates git build-base
RUN mkdir -p /go/src/taas
WORKDIR /go/src/taas
ADD . .
ENV GO111MODULE on
RUN go build .
CMD ["./taas"]
EXPOSE 4000

