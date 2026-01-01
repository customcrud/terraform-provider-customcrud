#!/usr/bin/env bash

set -eu

input="$(cat)"
secret_name="$(echo "$input" | jq --raw-output '.input.secret_name')"
project="$(echo "$input" | jq --raw-output '.input.project')"

gcloud secrets versions list $secret_name \
  --limit=1 \
  --project=$project \
  --format=json \
  --filter="state:ENABLED" \
| jq 'first'
