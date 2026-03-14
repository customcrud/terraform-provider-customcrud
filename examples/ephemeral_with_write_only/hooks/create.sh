#!/usr/bin/env bash

set -e
input="$(cat)"

content="$(echo "$input" | jq -r '.input.content')"
path="$(echo "$input" | jq -r '.input.path')"
echo -n "$content" > "$path"

jq -n --arg path "$path" '{id: $path, path: $path}'
