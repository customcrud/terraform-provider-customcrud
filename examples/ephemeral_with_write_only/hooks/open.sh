#!/usr/bin/env bash

set -e

jq -n --arg content "$(head -c 32 /dev/urandom | base64)" '{content: $content}'
