#!/usr/bin/env bash

set -e
input="$(cat)"

id="$(mktemp)"
content="$(echo "$input" | jq -r ".input.content")"
echo -n "$content" > "$id"

jq -n --arg id "$id" --arg content "$content" '{id: $id, content: $content}'