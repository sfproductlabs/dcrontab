#!/bin/bash
if [ -z "$1" ]
  then
    echo "No node supplied"
    exit 1
fi

if [ -z "$2" ]
  then
    echo "Skipping CA Generation"
  else
    openssl genrsa -out rootCa.key 4096
    openssl req -new -x509 -days 365 -key rootCa.key -out rootCa.crt -subj "/OU=SFPL"
fi

openssl genrsa -out dcrontab$1.key 1024
openssl req -new -key dcrontab$1.key -out dcrontab$1.csr -subj "/CN=dcrontab$1"
openssl x509 -req -days 365 -in dcrontab$1.csr -CA rootCa.crt -CAkey rootCa.key -set_serial 01 -out dcrontab$1.crt