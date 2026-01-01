#!/usr/bin/env bash

set -e

input="$(cat)"
id="$(echo "$input" | jq -r '.id')"
rm "$id"
