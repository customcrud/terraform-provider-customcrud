#!/usr/bin/env bash

set -eo pipefail

cat | ../../examples/file/hooks/create.sh | jq 'del(.content)'