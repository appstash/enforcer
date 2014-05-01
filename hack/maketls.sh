#!/bin/bash

go clean
go get && go build
./enforcer -tls -addr=0.0.0.0:443
