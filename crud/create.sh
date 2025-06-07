#!/usr/bin/env bash

set -e
INPUT="$(cat)"
1>&2 echo "[CREATE] INPUT_DUMP: $INPUT"

ID="$(mktemp)"
CONTENT="$(echo "$INPUT" | jq -r ".input.content")"
echo "$CONTENT" > "$ID"

jq -n --arg id "$ID" --arg content "$CONTENT" '{id: $id, content: $content}'