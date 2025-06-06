#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


set -e
INPUT="$(cat)"
1>&2 echo "[CREATE] INPUT_DUMP: $INPUT"
CONTENT="$(echo "$INPUT" | jq -r ".content")"
FILENAME="$(echo "$INPUT" | jq -r ".filename")"
echo "$CONTENT" > "$FILENAME"

echo "{\"id\": \"$FILENAME\", \"filename\": \"$FILENAME\", \"content\": \"$CONTENT\"}"