#!/usr/bin/env bash

jq -n '{id: 1, a: [1, "2", false, null, [{"b": 3}], [1, 2, 3]]}'