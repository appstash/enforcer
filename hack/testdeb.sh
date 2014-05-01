#!/usr/bin/env bash

dpkg -i ./bundles/0.1.0-dev/ubuntu/enforcer-0.1.0-dev_0.1.0-dev_amd64.deb
cd /opt/enforcer && ./enforcer-0.1.0-dev -addr=0.0.0.0:4321
