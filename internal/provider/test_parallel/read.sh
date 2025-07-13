#!/usr/bin/env bash
set -euo pipefail

input=$(cat)
id=$(echo "$input" | jq -r .id)
name=$(echo "$input" | jq -r .output.name)
jq -n --arg id "$id" --arg name "$name" '{id: $id, name: $name}'
