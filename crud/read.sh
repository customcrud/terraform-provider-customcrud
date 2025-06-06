#!/usr/bin/env bash

set -e

INPUT="$(cat)"
1>&2 echo "[READ] INPUT_DUMP: $INPUT"
FILENAME="$(echo "$INPUT" | jq -r ".id")"
ID="$(echo "$INPUT" | jq -r ".id")"

echo "{\"id\": \"$ID\", \"filename\": \"$FILENAME\", \"content\": \"$(cat "$FILENAME")\"}"
