#!/usr/bin/env bash

set -e

INPUT="$(cat)"
1>&2 echo "[DELETE] INPUT_DUMP: $INPUT"
FILENAME="$(echo "$INPUT" | jq -r '.filename')"
rm "$FILENAME"

echo "{}"