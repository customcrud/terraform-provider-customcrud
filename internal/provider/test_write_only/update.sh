#!/usr/bin/env bash

set -eo pipefail

cat | ../../examples/file/hooks/update.sh | jq 'del(.content)'