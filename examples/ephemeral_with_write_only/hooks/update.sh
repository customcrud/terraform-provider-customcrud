#!/usr/bin/env bash

set -e

input="$(cat)"

id="$(echo "$input" | jq -r '.id')"
previous_path="$(echo "$input" | jq -r '.output.path')"
path="$(echo "$input" | jq -r '.input.path')"

# Move the file, we want to preserve the existing urandom value
mv "$previous_path" "$path"

jq -n --arg id "$id" --arg path "$path" '{id: $id, path: $path}'
