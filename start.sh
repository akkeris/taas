#!/bin/sh

if [ -f /var/run/secrets/kubernetes.io/serviceaccount/ca.crt ]
then
   cat  /var/run/secrets/kubernetes.io/serviceaccount/ca.crt >> /etc/ssl/certs/ca-certificates.crt
fi

if [[ -n "$1" && "$1" == "cron" ]]
then
    /go/src/taas/taas --cron_worker
else
   /go/src/taas
fi
