#!/bin/bash

LOG_FILE="$1"
CSV_FILE="$2"

echo "state_root,epoch,miner,penalty,gas_reward,win_count,exit_code" > "$CSV_FILE"

cat "$LOG_FILE" |rg "FRRIST" | rg "ApplyBlocks" | jq ".msg" | awk '{print $5 $7 $9 $11 $13 $15 $17}' | cut -d '"' -f 1 >> "$CSV_FILE"
