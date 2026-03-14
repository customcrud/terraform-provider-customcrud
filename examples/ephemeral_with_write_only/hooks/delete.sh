#!/usr/bin/env bash

set -e

rm -f "$(cat | jq -r '.output.path')"
