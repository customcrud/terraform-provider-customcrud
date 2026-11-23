#!/usr/bin/env bash

set -euo pipefail

input="$(cat)"
name="$(echo "$input" | jq -r .input.name)"

lockfile="/tmp/${name}.lock"
# noclobber opens with O_EXCL; uutils mkdir (Ubuntu 26.04) is not exclusive under a race
if ! (set -o noclobber; : > "$lockfile") 2>/dev/null; then
  jq -n --arg name "$name" '{error: "lock [\($name)] already held"}' >&2
  exit 1
fi

trap 'rm -f "$lockfile"' EXIT

# Ensures terraform can create the resource fast enough to create a collision if
# parallism is enabled
sleep 0.1

id="$(head -c8 /dev/urandom | xxd -p)"
jq -n --arg id "$id" --arg name "$name" '{id: $id, name: $name}'
