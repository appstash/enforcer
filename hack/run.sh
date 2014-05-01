#!/usr/bin/env bash

serf agent &
./bundles/0.2.0-dev/binary/enforcer-0.2.0-dev -d -debug -addr=0.0.0.0:4321
