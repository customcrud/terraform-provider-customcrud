#!/usr/bin/env bash

set -e

INPUT="$(cat)"
1>&2 echo "[UPDATE] INPUT_DUMP: $INPUT"
ID="$(echo "$INPUT" | jq -r ".id")"
FILENAME="$(echo "$INPUT" | jq -r ".filename")"
CONTENT="$(echo "$INPUT" | jq -r ".content")"

echo "$CONTENT" > "$FILENAME"

echo "{\"id\": \"$ID\", \"filename\": \"$FILENAME\", \"content\": \"$CONTENT\"}"