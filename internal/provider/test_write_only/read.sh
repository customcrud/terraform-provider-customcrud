#!/usr/bin/env bash

set -eo pipefail

cat | ../../examples/file/hooks/read.sh | jq 'del(.content)'