#!/usr/bin/env bash

read -r input

echo "Failed to delete resource: Resource is locked" >&2

exit 7 