#!/bin/bash
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key -out ca.crt
openssl genrsa -out node1.key 1024
openssl req -new -key node1.key -out node1.csr
openssl x509 -req -days 365 -in node1.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out node1.crt

