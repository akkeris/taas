FROM golang:1.8-alpine
RUN apk update
RUN apk add openssl ca-certificates git
RUN mkdir -p /go/src/taas
ADD server.go  /go/src/taas/server.go
ADD build.sh /build.sh
RUN chmod +x /build.sh
ADD notifications /go/src/taas/notifications
ADD alamo /go/src/taas/alamo
ADD structs /go/src/taas/structs
ADD diagnosticlogs /go/src/taas/diagnosticlogs
ADD diagnostics /go/src/taas/diagnostics
ADD hooks /go/src/taas/hooks
ADD pipelines /go/src/taas/pipelines
ADD githubapi /go/src/taas/githubapi
ADD dbstore /go/src/taas/dbstore
RUN /build.sh
CMD ["/go/src/taas/server"]
EXPOSE 4000

