FROM golang:1.8-alpine
RUN apk update
RUN apk add openssl ca-certificates git
RUN mkdir -p /go/src/taas
ADD server.go  /go/src/taas/server.go
ADD build.sh /build.sh
RUN chmod +x /build.sh
ADD notifications /go/src/taas/notifications
ADD jobs /go/src/taas/jobs
ADD structs /go/src/taas/structs
ADD diagnosticlogs /go/src/taas/diagnosticlogs
ADD diagnostics /go/src/taas/diagnostics
ADD hooks /go/src/taas/hooks
ADD pipelines /go/src/taas/pipelines
ADD githubapi /go/src/taas/githubapi
ADD dbstore /go/src/taas/dbstore
ADD artifacts /go/src/taas/artifacts
ADD create.sql /go/src/taas/create.sql
RUN /build.sh
ADD start.sh /start.sh
RUN chmod +x /start.sh
CMD ["/start.sh"]
WORKDIR /go/src/taas
EXPOSE 4000

