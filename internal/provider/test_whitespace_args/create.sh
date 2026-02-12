#!/usr/bin/env bash

set -e

input="$(cat)"
id="$(echo "$input" | jq -r '.input.name')"

# Build a JSON array from all positional arguments
args_json=$(printf '%s\n' "$@" | jq --raw-input . | jq --slurp .)

jq --null-input --arg id "$id" --argjson args "$args_json" '{id: $id, args: $args}'
