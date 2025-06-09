#!/usr/bin/env bash

set -e

INPUT="$(cat)"
1>&2 echo "[UPDATE] INPUT_DUMP: $INPUT"
ID="$(echo "$INPUT" | jq -r ".id")"
CONTENT="$(echo "$INPUT" | jq -r ".input.content")"

echo "$CONTENT" > "$ID"
jq -n --arg id "$ID" --arg content "$(cat "$ID")" '{id: $id, content: $content}'
