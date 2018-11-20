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
cd /go/src/taas
go build server.go

