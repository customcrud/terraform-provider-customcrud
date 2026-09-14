#!/usr/bin/env bash

read -r input

echo "Failed to create resource: Permission denied ($input)" >&2

exit 13
