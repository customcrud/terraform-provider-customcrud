#!/usr/bin/env bash

set -e

input="$(cat)"
id="$(echo "$input" | jq -r ".id")"
content="$(echo "$input" | jq -r ".input.content")"

echo -n "$content" > "$id"
jq -n --arg id "$id" --arg content "$(cat "$id")" '{id: $id, content: $content}'
