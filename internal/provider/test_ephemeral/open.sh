#!/usr/bin/env bash

set -e

input="$(cat)"

# Extract marker_file from input if provided
marker_file="$(echo "$input" | jq -r '.input.marker_file // empty')"
if [[ -n "$marker_file" ]]; then
  echo "open" > "$marker_file"
fi

# Generate output based on input
name="$(echo "$input" | jq -r '.input.name // "test"')"
timestamp="$(date +%s)"

jq -n --arg name "$name" --arg ts "$timestamp" '{name: $name, timestamp: $ts, status: "opened"}'
