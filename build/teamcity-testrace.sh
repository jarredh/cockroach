#!/usr/bin/env bash
set -euxo pipefail
build/builder.sh make testrace TESTFLAGS='-v' 2>&1 | go-test-teamcity
