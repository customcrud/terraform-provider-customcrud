#!/usr/bin/env bash

set -e

INPUT="$(cat)"
1>&2 echo "[DELETE] INPUT_DUMP: $INPUT"
ID="$(echo "$INPUT" | jq -r '.id')"
rm "$ID"