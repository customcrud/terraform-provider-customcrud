#!/usr/bin/env bash

set -e

INPUT="$(cat || exit 22)"
1>&2 echo "[READ] INPUT_DUMP: $INPUT"
ID="$(echo "$INPUT" | jq -r ".id")"

jq -n --arg id "$ID" --arg content "$(cat "$ID")" '{id: $id, content: $content}'
