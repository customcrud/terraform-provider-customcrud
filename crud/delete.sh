#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


set -e

INPUT="$(cat)"
1>&2 echo "[DELETE] INPUT_DUMP: $INPUT"
FILENAME="$(echo "$INPUT" | jq -r '.filename')"
rm "$FILENAME"

echo "{}"