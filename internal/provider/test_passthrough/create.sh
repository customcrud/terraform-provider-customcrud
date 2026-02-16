#!/usr/bin/env bash
# Echoes input fields as output. If input contains remove_from_output (array of keys),
# those keys (and remove_from_output itself) are removed from the output.
input=$(cat)
echo "$input" | jq '
  {id: "test-passthrough"} + (.input // {})
  | . as $base
  | ($base.remove_from_output // []) as $keys
  | $base | del(.remove_from_output)
  | reduce ($keys[]) as $k (.; del(.[$k]))
'
