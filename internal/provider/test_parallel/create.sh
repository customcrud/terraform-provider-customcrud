#!/usr/bin/env bash

set -euo pipefail

input="$(cat)"
name="$(echo "$input" | jq -r .input.name)"

lockdir="/tmp/${name}.lock"
if ! mkdir "$lockdir" 2>/dev/null; then
  jq -n --arg name "$name" '{error: "lock [\($name)] already held"}' >&2
  exit 1
fi

trap 'rmdir "$lockdir"' EXIT

# Ensures terraform can create the resource fast enough to create a collision if
# parallism is enabled
sleep 0.1

id="$(head -c8 /dev/urandom | xxd -p)"
jq -n --arg id "$id" --arg name "$name" '{id: $id, name: $name}'
