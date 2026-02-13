#!/usr/bin/env bash

set -e

input="$(cat)"

# Extract marker_file from input if provided
marker_file="$(echo "$input" | jq -r '.input.marker_file // empty')"
if [[ -n "$marker_file" ]]; then
  echo "renew" >> "$marker_file"
fi

# Renew doesn't produce output per Terraform spec, just exit successfully
exit 0
