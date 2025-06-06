#!/usr/bin/env bash

set -e
INPUT="$(cat)"
1>&2 echo "[CREATE] INPUT_DUMP: $INPUT"
CONTENT="$(echo "$INPUT" | jq -r ".content")"
FILENAME="$(echo "$INPUT" | jq -r ".filename")"
echo "$CONTENT" > "$FILENAME"

echo "{\"id\": \"$FILENAME\", \"filename\": \"$FILENAME\", \"content\": \"$CONTENT\"}"