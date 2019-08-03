#!/bin/sh

cd /go/src
go get  "github.com/nu7hatch/gouuid"
go get  "github.com/go-martini/martini"
go get  "github.com/martini-contrib/binding"
go get  "github.com/martini-contrib/render"
go get  "github.com/lib/pq"
go get  "github.com/davecgh/go-spew/spew"
go get  "github.com/bitly/go-simplejson"
go get  "github.com/akkeris/vault-client"
go get  "github.com/aws/aws-sdk-go/aws"
go get  "github.com/aws/aws-sdk-go/aws/awserr"
go get  "github.com/aws/aws-sdk-go/aws/session"
go get  "github.com/aws/aws-sdk-go/service/s3"
go get  "github.com/mattn/go-shellwords"
go get  "github.com/bsm/sarama-cluster"
cd /go/src/taas
go build server.go

