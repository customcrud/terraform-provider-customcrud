#!/usr/bin/env bash

set -eo pipefail

input="$(cat)"
target="$(echo "$input" | jq -r .input.target)"

jq -n --arg target $target '{target: $target | tonumber, id: "1"}'
