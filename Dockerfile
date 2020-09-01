FROM golang:1.12-alpine
RUN apk update
RUN apk add openssl ca-certificates git build-base
RUN mkdir -p /go/src/taas
WORKDIR /go/src/taas
ADD . .
ENV GO111MODULE on
RUN go build .
RUN chmod +x ./start.sh
CMD ["./start.sh"]
EXPOSE 4000