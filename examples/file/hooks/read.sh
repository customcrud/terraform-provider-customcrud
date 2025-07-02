#!/usr/bin/env bash

set -e

INPUT="$(cat)"
1>&2 echo "[READ] INPUT_DUMP: $INPUT"
ID="$(echo "$INPUT" | jq -r ".id")"
content=$(cat "$ID") || exit 22

jq -n --arg id "$ID" --arg content "$content" '{id: $id, content: $content}'
