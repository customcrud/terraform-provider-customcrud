#!/usr/bin/env bash

set -e

input="$(cat)"

id="$(echo "$input" | jq -r '.id // .input.path')"
content=$(cat "$id") || exit 22

jq -n --arg id "$id" --arg content "$content" '{id: $id, content: $content}'
